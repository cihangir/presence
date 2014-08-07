package presence

import (
	"sync"
	"testing"
	"time"
)

func initFaultTolerantRedis(t *testing.T) *Session {

	backend, err := NewFaultTolerantRedis("localhost:6379", 10, time.Second*1)
	if err != nil {
		t.Fatal(err)
	}

	ses, err := New(backend)
	if err != nil {
		t.Fatal(err)
	}

	return ses
}

func TestFaultTolerantRedisOffline(t *testing.T) {
	// sleep for evicting keys
	defer time.Sleep(time.Second * 2)

	s := initFaultTolerantRedis(t)
	defer s.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	offlines := []string{"1", "2", "3", "4", "5"}
	requiredEventCount := len(offlines)

	offlineCount := 0
	go func() {
		defer wg.Done()
		e := s.ListenStatusChanges()
		for {
			requiredEventCount--
			select {
			case status, ok := <-e:
				if !ok {
					return
				}
				if status.Status == Offline {
					offlineCount++
				}
			case <-time.After(time.Second * 1):
				t.Fatal("did not get required messages")
			}
			if requiredEventCount == 0 {
				return
			}
		}
	}()

	err := s.Offline(offlines...)
	if err != nil {
		t.Error("error should be nil while setting users offline %s", err)
	}

	wg.Wait()
	if offlineCount != len(offlines) {
		t.Fatal("offline count is not %d, it is %d", len(offlines), offlineCount)
	}
}

func TestFaultTolerantRedisOnline(t *testing.T) {
	// sleep for evicting keys
	defer time.Sleep(time.Second * 2)

	s := initFaultTolerantRedis(t)
	defer s.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	onlines := []string{"1", "2", "3", "4", "5"}
	requiredEventCount := len(onlines)

	onlineCount := 0
	go func() {
		defer wg.Done()
		e := s.ListenStatusChanges()
		for {
			requiredEventCount--
			select {
			case status, ok := <-e:
				if !ok {
					return
				}
				if status.Status == Online {
					onlineCount++
				}
			case <-time.After(time.Second * 1):
				t.Fatal("did not get required messages")

			}
			if requiredEventCount == 0 {
				return
			}
		}
	}()

	err := s.Online(onlines...)
	if err != nil {
		t.Error("error should be nil while setting users online %s", err)
	}

	wg.Wait()
	if onlineCount != len(onlines) {
		t.Fatal("online count is not %d, it is %d", len(onlines), onlineCount)
	}

}

func TestFaultTolerantRedisStatus(t *testing.T) {
	s := initFaultTolerantRedis(t)
	defer s.Close()

	_, err := s.Status("1", "2", "3", "4", "5")
	if err != nil {
		t.Error("error should be nil while querying the statuses %s", err)
	}
}

func TestFaultTolerantRedisTTL(t *testing.T) {
	s := initFaultTolerantRedis(t)

	var wg sync.WaitGroup
	wg.Add(1)

	onlines := []string{"6", "7", "8", "9", "10"}

	// first pass for onlines
	// second pass for offlines
	requiredEventCount := len(onlines) * 2

	go func() {
		defer wg.Done()

		// close connection after 3 seconds, we should have all the events at
		// this point
		time.AfterFunc(time.Second*3, func() {
			s.Close()
		})

		for {
			select {
			case _, ok := <-s.ListenStatusChanges():
				if !ok {
					return
				} else {
					requiredEventCount--
				}
			}
		}
	}()

	err := s.Online(onlines...)
	if err != nil {
		t.Errorf("error should be nil while setting users online %s", err)
	}
	wg.Wait()

	if requiredEventCount != 0 {
		t.Errorf("decreased event count is not 0, it is %d", requiredEventCount)
	}
}
