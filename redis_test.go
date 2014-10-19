package presence

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

var nextID chan string
var testTimeoutDuration = time.Second * 1

func init() {
	nextID = make(chan string)
	go func() {
		id := 0
		for {
			id++
			nextID <- "id" + strconv.Itoa(id)
		}
	}()
}

func initPresence() (*Session, error) {
	connStr := os.Getenv("REDIS_URI")
	if connStr == "" {
		connStr = "localhost:6379"
	}

	backend, err := NewRedis(connStr, 10, testTimeoutDuration)
	if err != nil {
		return nil, err
	}

	// adjust config for redis instance
	c := backend.(*Redis).redis.Pool().Get()
	if _, err := c.Do("CONFIG", "SET", "notify-keyspace-events", "Ex$"); err != nil {
		return nil, err
	}

	ses, err := New(backend)
	if err != nil {
		return nil, err
	}

	return ses, nil
}

func withConn(f func(s *Session)) error {
	s, err := initPresence()
	if err != nil {
		return err
	}

	f(s)

	return s.Close()
}

func TestOnline(t *testing.T) {
	err := withConn(func(s *Session) {
		id := <-nextID
		if err := s.Online(id); err != nil {
			t.Fatalf("non existing id can be set as online, but got err: %s", err.Error())
		}

		if err := s.Online(id); err != nil {
			t.Fatalf("existing id can be set as online again, but got err: %s", err.Error())
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestOnlineMultiple(t *testing.T) {
	err := withConn(func(s *Session) {
		ids := []string{<-nextID, <-nextID}
		if err := s.Online(ids...); err != nil {
			t.Fatalf("non existing ids can be set as online, but got err: %s", err.Error())
		}

		if err := s.Online(ids...); err != nil {
			t.Fatalf("existing ids can be set as online again, but got err: %s", err.Error())
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestOffline(t *testing.T) {
	err := withConn(func(s *Session) {
		id := <-nextID
		if err := s.Offline(id); err != nil {
			t.Fatalf("non existing id can be set as offline, but got err: %s", err.Error())
		}

		if err := s.Offline(id); err != nil {
			t.Fatalf("existing id can be set as offline again, but got err: %s", err.Error())
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestOfflineMultiple(t *testing.T) {
	err := withConn(func(s *Session) {
		ids := []string{<-nextID, <-nextID}
		if err := s.Offline(ids...); err != nil {
			t.Fatalf("non existing ids can be set as offline, but got err: %s", err.Error())
		}

		if err := s.Offline(ids...); err != nil {
			t.Fatalf("existing ids can be set as offline again, but got err: %s", err.Error())
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusOnline(t *testing.T) {
	err := withConn(func(s *Session) {
		id := <-nextID
		if err := s.Online(id); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(id)
		if err != nil {
			t.Fatal(err)
		}

		res := status[0]

		if res.Status != Online {
			t.Fatalf("%s should be %s, but it is %s", res.ID, Online, res.Status)
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusOffline(t *testing.T) {
	err := withConn(func(s *Session) {

		id := <-nextID
		if err := s.Offline(id); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(id)
		if err != nil {
			t.Fatal(err)
		}

		res := status[0]
		if res.Status != Offline {
			t.Fatalf("%s should be %s, but it is %s", res.ID, Offline, res.Status)
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusMultiAllOnline(t *testing.T) {
	err := withConn(func(s *Session) {
		ids := []string{<-nextID, <-nextID}
		// mark all of them as online first
		if err := s.Online(ids...); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(ids...)
		if err != nil {
			t.Fatal(err)
		}

		for _, res := range status {
			if res.Status != Online {
				t.Fatalf("%s should be %s, but it is %s", res.ID, Online, res.Status)
			}
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusMultiAllOffline(t *testing.T) {
	err := withConn(func(s *Session) {
		ids := []string{<-nextID, <-nextID}

		// mark all of them as offline first
		if err := s.Offline(ids...); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(ids...)
		if err != nil {
			t.Fatal(err)
		}

		for _, res := range status {
			if res.Status != Offline {
				t.Fatalf("%s should be %s, but it is %s", res.ID, Offline, res.Status)
			}
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusMultiMixed(t *testing.T) {
	err := withConn(func(s *Session) {
		onlineID := <-nextID
		offlineID := <-nextID

		ids := []string{onlineID, offlineID}

		if err := s.Online(onlineID); err != nil {
			t.Fatal(err)
		}

		if err := s.Offline(offlineID); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(ids...)
		if err != nil {
			t.Fatal(err)
		}

		if status[0].Status != Online {
			t.Fatalf("%s should be %s, but it is %s", status[0].ID, Online, status[0].Status)
		}
		if status[1].Status != Offline {
			t.Fatalf("%s should be %s, but it is %s", status[1].ID, Offline, status[0].Status)
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusOrder(t *testing.T) {
	err := withConn(func(s *Session) {
		ids := []string{<-nextID, <-nextID}

		if err := s.Online(ids...); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(ids...)
		if err != nil {
			t.Fatal(err)
		}

		for i := range status {
			if status[i].ID != ids[i] {
				t.Fatalf("%dth status should be %s, but it is %s", i, ids[i], status[i].ID)
			}
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusLen(t *testing.T) {
	err := withConn(func(s *Session) {
		ids := []string{<-nextID, <-nextID}

		if err := s.Online(ids...); err != nil {
			t.Fatal(err)
		}

		status, err := s.Status(ids...)
		if err != nil {
			t.Fatal(err)
		}

		if len(status) != len(ids) {
			t.Fatalf("Status response len should be: %d, but got: %d", len(ids), len(status))
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusWithTimeout(t *testing.T) {
	err := withConn(func(s *Session) {
		id := <-nextID
		if err := s.Online(id); err != nil {
			t.Fatal(err)
		}

		// sleep until expiration
		time.Sleep(testTimeoutDuration * 2)

		// get the status of the id
		status, err := s.Status(id)
		if err != nil {
			t.Fatal(err)
		}

		res := status[0]
		if res.Status != Offline {
			t.Fatalf("%s should be %s, but it is %s", res.ID, Offline, res.Status)
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestSubscriptions(t *testing.T) {
	err := withConn(func(s *Session) {

		ids := []string{<-nextID, <-nextID, <-nextID}

		// sleep until expiration
		time.Sleep(testTimeoutDuration * 2)

		onlineCount := 0
		offlineCount := 0
		go func() {
			for event := range s.ListenStatusChanges() {
				switch event.Status {
				case Online:
					onlineCount++
				case Offline:
					offlineCount++
				}
			}
		}()

		err := s.Online(ids...)
		if err != nil {
			t.Fatal(err)
		}

		// sleep until expiration
		time.Sleep(testTimeoutDuration * 2)

		if onlineCount != len(ids) {
			t.Fatal(
				fmt.Errorf("online count should be: %d, but it is: %d", len(ids), onlineCount),
			)
		}

		if offlineCount != len(ids) {
			t.Fatal(
				fmt.Errorf("offline count should be: %d, but it is: %d", len(ids), offlineCount),
			)
		}
	})

	if err != nil {
		t.Fatal(err)
	}
}
