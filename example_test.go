package presence

import (
	"fmt"
	"os"
	"time"
)

func ExampleRedis_listenStatusChanges() {
	connStr := os.Getenv("REDIS_URI")
	if connStr == "" {
		connStr = "localhost:6379"
	}

	backend, err := NewRedis(connStr, 10, time.Second*1)
	if err != nil {
		fmt.Println(err.Error())
	}

	// adjust config for redis instance
	c := backend.(*Redis).redis.Pool().Get()
	if _, err := c.Do("CONFIG", "SET", "notify-keyspace-events", "Ex$"); err != nil {
		fmt.Println(err)
	}

	if err := c.Close(); err != nil {
		fmt.Println(err)
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
