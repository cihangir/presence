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

type Event struct {
	Id     string
	Status Status
}

const RedisencePrefix = "redisence"

type Session struct {
	redis            *redis.RedisSession
	inactiveDuration time.Duration
	becameOffline    string
	becameOnline     string
}

func New(server string, db int, inactiveDuration time.Duration) (*Session, error) {
	redis, err := redis.NewRedisSession(&redis.RedisConf{Server: server, DB: db})
	if err != nil {
		return nil, err
	}
	redis.SetPrefix(RedisencePrefix)

	return &Session{
		redis:            redis,
		becameOffline:    fmt.Sprintf("__keyevent@%d__:expired", db),
		becameOnline:     fmt.Sprintf("__keyevent@%d__:set", db),
		inactiveDuration: inactiveDuration,
	}, nil
}

// Whenever we get any prob from a client
// we should call this function
func (s *Session) Ping(id string) error {
	if s.redis.Expire(id, s.inactiveDuration) == nil {
		return nil
	}
	return s.redis.Setex(id, s.inactiveDuration, id)
}

func (s *Session) Status(id string) Status {
	if s.redis.Exists(id) {
		return Online
	}

	return Offline
}

func (s *Session) createEvent(n gredis.PMessage) Event {
	e := Event{}

	switch n.Pattern {
	case s.becameOffline:
		e.Id = string(n.Data[len(RedisencePrefix)+1:])
		e.Status = Offline
	case s.becameOnline:
		e.Id = string(n.Data[len(RedisencePrefix)+1:])
		e.Status = Online
	default:
		//ignore other events
	}

	return e
}

func (s *Session) ListenStatusChanges(events chan Event) {
	psc := s.redis.CreatePubSubConn()

	psc.PSubscribe(s.becameOnline, s.becameOffline)

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
		psc.PUnsubscribe(s.becameOffline, s.becameOnline)
		psc.Close()
		events <- Event{Status: Closed}
	}()
}
