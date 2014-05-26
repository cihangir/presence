package redisence

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	gredis "github.com/garyburd/redigo/redis"
	"github.com/koding/redis"
)

type Status int

const (
	Offline Status = iota
	Online
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
func (s *Session) Ping(ids ...string) error {
	if len(ids) == 1 {
		// if member exits increase ttl
		if s.redis.Expire(ids[0], s.inactiveDuration) == nil {
			return nil
		}

		// if member doesnt exist set it
		return s.redis.Setex(ids[0], s.inactiveDuration, ids[0])
	}

	existance, err := s.sendMultiExpire(ids)
	if err != nil {
		return err
	}

	return s.sendMultiSetIfRequired(ids, existance)
}

func (s *Session) sendMultiSetIfRequired(ids []string, existance []int) error {
	if len(ids) != len(existance) {
		return fmt.Errorf("Length is not same Ids: %d Existance: %d", len(ids), len(existance))
	}

	// cache inactive duration as string
	seconds := strconv.Itoa(int(s.inactiveDuration.Seconds()))

	// get one connection from pool
	c := s.redis.Pool().Get()

	// item count for non-existent members
	notExistsCount := 0

	for i, exists := range existance {
		// `0` means, member doesnt exists in presence system
		if exists != 0 {
			continue
		}

		// init multi command lazily
		if notExistsCount == 0 {
			c.Send("MULTI")
		}

		notExistsCount++
		c.Send("SETEX", s.redis.AddPrefix(ids[i]), seconds, ids[i])
	}

	// execute multi command if only we flushed some to connection
	if notExistsCount != 0 {
		// ignore values
		if _, err := c.Do("EXEC"); err != nil {
			return err
		}
	}

	// do not forget to close the connection
	if err := c.Close(); err != nil {
		return err
	}

	return nil
}

func (s *Session) sendMultiExpire(ids []string) ([]int, error) {
	// cache inactive duration as string
	seconds := strconv.Itoa(int(s.inactiveDuration.Seconds()))

	// get one connection from pool
	c := s.redis.Pool().Get()

	// init multi command
	c.Send("MULTI")

	// send expire command for all members
	for _, id := range ids {
		c.Send("EXPIRE", s.redis.AddPrefix(id), seconds)
	}

	// execute command
	r, err := c.Do("EXEC")
	if err != nil {
		return make([]int, 0), err
	}

	// close connection
	if err := c.Close(); err != nil {
		return make([]int, 0), err
	}

	values, err := s.redis.Values(r)
	if err != nil {
		return make([]int, 0), err
	}

	res := make([]int, len(values))
	for i, value := range values {
		res[i], err = s.redis.Int(value)
		if err != nil {
			return make([]int, 0), err
		}

	}

	return res, nil
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
