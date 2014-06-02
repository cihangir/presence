package redisence

import (
	"fmt"
	"time"
)

func main() {
	s, err := New("localhost:6379", 10, time.Second*1)
	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(time.Second * 1)
		s.Online("id")
		time.Sleep(time.Second * 1)
		s.Online("id")
		s.Online("id2")
		s.Online("id2")
		s.Online("id2")
		s.Online("id3")
	}()

	for event := range s.ListenStatusChanges() {
		switch event.Status {
		case Online:
			fmt.Println(event)
		case Offline:
			fmt.Println(event)
		}
	}
}
