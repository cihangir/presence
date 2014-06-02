package redisence

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func initRedisence(t *testing.T) *Session {
	ses, err := New("localhost:6379", 10, time.Second*1)
	if err != nil {
		t.Fatal(err)
	}
	conn := ses.redis.Pool().Get()
	conn.Do("CONFIG", "SET", "notify-keyspace-events Ex$")
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	return ses
}

func TestInitialization(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()
}

func TestSinglePing(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Online("id"); err != nil {
		t.Fatal(err)
	}
}

func TestMultiPing(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Online("id", "id2"); err != nil {
		t.Fatal(err)
	}
}

func TestOnlineStatus(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	id := "id3"
	if err := s.Online(id); err != nil {
		t.Fatal(err)
	}

	status, err := s.Status(id)
	if err != nil {
		t.Fatal(err)
	}

	if status.Status != Online {
		t.Fatal(errors.New("User should be active"))
	}
}

func TestOfflineStatus(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	id := "id4"
	if err := s.Online(id); err != nil {
		t.Fatal(err)
	}

	status, err := s.Status("id5")
	if err != nil {
		t.Fatal(err)
	}

	if status.Status != Offline {
		t.Fatal(errors.New("User should be offline"))
	}
}

func TestMultiStatusAllOnline(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Online("id6", "id7"); err != nil {
		t.Fatal(err)
	}

	status, err := s.MultipleStatus([]string{"id6", "id7"})
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range status {
		if st.Status != Online {
			t.Fatal(errors.New("User should be active"))
		}
	}
}

func TestMultiStatusAllOffline(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Online("id8", "id9"); err != nil {
		t.Fatal(err)
	}

	status, err := s.MultipleStatus([]string{"id10", "id11"})
	if err != nil {
		t.Fatal(err)
	}

	for _, st := range status {
		if st.Status != Offline {
			t.Fatal(errors.New("User should be offline"))
		}
	}
}

func TestStatusWithTimeout(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	id := "12"
	if err := s.Online(id); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 2)
	status, err := s.Status(id)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status == Online {
		t.Fatal(errors.New("User should not be active"))
	}
}

func TestSubscriptions(t *testing.T) {
	t.Skip("Skipped to travis")
	s := initRedisence(t)

	// wait for all keys to expire
	time.Sleep(time.Second * 1)

	id1 := "13"
	id2 := "14"
	id3 := "15"

	time.AfterFunc(time.Second*5, func() {
		err := s.Close()
		if err != nil {
			t.Fatal(err)
		}
	})

	time.AfterFunc(time.Second*1, func() {
		err := s.Online(id1, id2, id3)
		if err != nil {
			t.Fatal(err)
		}
		// err = s.Offline(id1, id2, id3)
		// if err != nil {
		// 	t.Fatal(err)
		// }
	})

	onlineCount := 0
	offlineCount := 0
	for event := range s.ListenStatusChanges() {
		switch event.Status {
		case Online:
			onlineCount++
		case Offline:
			offlineCount++
		}
	}

	if onlineCount != 3 {
		t.Fatal(
			errors.New(
				fmt.Sprintf("online count should be 3 it is %d", onlineCount),
			),
		)
	}

	if offlineCount != 3 {
		t.Fatal(
			errors.New(
				fmt.Sprintf("offline count should be 3 it is %d", offlineCount),
			),
		)
	}
}

func TestJustMultiOffline(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Offline("id16", "id17"); err != nil {
		t.Fatal(err)
	}
}

func TestMultiOnlineAndOfflineTogether(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Online("id18", "id19"); err != nil {
		t.Fatal(err)
	}
	if err := s.Offline("id18", "id19"); err != nil {
		t.Fatal(err)
	}
}

func TestMultiOfflineWithMultiStatus(t *testing.T) {
	s := initRedisence(t)
	defer s.Close()

	if err := s.Online("id20", "id21"); err != nil {
		t.Fatal(err)
	}
	if err := s.Offline("id20", "id21"); err != nil {
		t.Fatal(err)
	}
	status, err := s.MultipleStatus([]string{"id20", "id21"})
	if err != nil {
		t.Fatal(err)
	}

	for _, st := range status {
		if st.Status != Offline {
			t.Fatal(errors.New("User should be offline"))
		}
	}
}
