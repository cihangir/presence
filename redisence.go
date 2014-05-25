package redisence

import (
	"fmt"
	"sync"
	"time"

	gredis "github.com/garyburd/redigo/redis"
	"github.com/koding/redis"
)

type Status int

const (
	Online Status = iota
	Offline
	Closed
)

// Event is the data type for
// occuring events in the system
type Event struct {
	// Id is the given key by the application
	Id string

	// Status holds the changing type of event
	Status Status
}

// Prefix for redisence package
const RedisencePrefix = "redisence"

// Session holds the required connection data for redis
type Session struct {
	// main redis connection
	redis *redis.RedisSession

	// inactiveDuration specifies no-probe allowance time
	inactiveDuration time.Duration

	// receiving offline events pattern
	becameOfflinePattern string

	// receiving online events pattern
	becameOnlinePattern string
}

func New(server string, db int, inactiveDuration time.Duration) (*Session, error) {
	redis, err := redis.NewRedisSession(&redis.RedisConf{Server: server, DB: db})
	if err != nil {
		return nil, err
	}
	redis.SetPrefix(RedisencePrefix)

	return &Session{
		redis:                redis,
		becameOfflinePattern: fmt.Sprintf("__keyevent@%d__:expired", db),
		becameOnlinePattern:  fmt.Sprintf("__keyevent@%d__:set", db),
		inactiveDuration:     inactiveDuration,
	}, nil
}

// Ping resets the expiration time for any given key
// if key doesnt exists, it means user is now online
// Whenever application gets any prob from a client
// should call this function
func (s *Session) Ping(id string) error {
	if s.redis.Expire(id, s.inactiveDuration) == nil {
		return nil
	}
	return s.redis.Setex(id, s.inactiveDuration, id)
}

// Status returns the current status a key from system
// TODO use variadic function arguments
func (s *Session) Status(id string) Status {
	// to-do use MGET instead of exists
	if s.redis.Exists(id) {
		return Online
	}

	return Offline
}

// createEvent Creates the event with the required properties
func (s *Session) createEvent(n gredis.PMessage) Event {
	e := Event{}

	switch n.Pattern {
	case s.becameOfflinePattern:
		e.Id = string(n.Data[len(RedisencePrefix)+1:])
		e.Status = Offline
	case s.becameOnlinePattern:
		e.Id = string(n.Data[len(RedisencePrefix)+1:])
		e.Status = Online
	default:
		//ignore other events
	}

	return e
}

// ListenStatusChanges pubscribes to the redis and
// gets online and offline status changes from it
func (s *Session) ListenStatusChanges(events chan Event) {
	psc := s.redis.CreatePubSubConn()

	psc.PSubscribe(s.becameOnlinePattern, s.becameOfflinePattern)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			switch n := psc.Receive().(type) {
			case gredis.PMessage:
				events <- s.createEvent(n)
			case error:
				fmt.Printf("error: %v\n", n)
				return
			}
		}
	}()

	// avoid lock
	go func() {
		wg.Wait()
		psc.PUnsubscribe(s.becameOfflinePattern, s.becameOnlinePattern)
		psc.Close()
		events <- Event{Status: Closed}
	}()
}
