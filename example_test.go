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

	events := make(chan Event, 10)

	go s.ListenStatusChanges(events)

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

	for event := range events {
		switch event.Status {
		case Online:
			fmt.Println(event)
		case Offline:
			fmt.Println(event)
		case Closed:
			close(events)
			fmt.Println(event)
			return
		}
	}
}
