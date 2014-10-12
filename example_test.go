package presence

import (
	"fmt"
	"time"
)

func ExampleListenStatusChanges() {
	backend, err := NewRedis("localhost:6379", 10, time.Second*1)
	if err != nil {
		fmt.Println(err.Error())
	}

	s, err := New(backend)
	if err != nil {
		fmt.Println(err.Error())
	}

	go func() {
		for event := range s.ListenStatusChanges() {
			switch event.Status {
			case Online:
				fmt.Println(event)
			case Offline:
				fmt.Println(event)
			}
		}
	}()

	go func() {
		s.Online("id")
	}()

	// wait for events
	<-time.After(time.Second * 2)

	// Output:
	// {id ONLINE}
	// {id OFFLINE}
}
