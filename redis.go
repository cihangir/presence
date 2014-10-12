// Package presence provides simple user presence system
package presence

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	gredis "github.com/garyburd/redigo/redis"
	"github.com/koding/redis"
)

// Prefix for redisence package
var RedisencePrefix = "redisence"

// Redis holds the required connection data for redis
type Redis struct {
	// main redis connection
	redis *redis.RedisSession

	// inactiveDuration specifies no-probe allowance time
	inactiveDuration time.Duration

	// receiving offline events pattern
	becameOfflinePattern string

	// receiving online events pattern
	becameOnlinePattern string

	// errChan pipe all errors  the this channel
	errChan chan error

	// closed holds the status of connection
	closed bool

	//psc holds the pubsub channel if opened
	psc *gredis.PubSubConn

	// holds event channel
	events chan Event

	// lock for Redis struct
	mu sync.Mutex
}

// NewRedis creates a Redis for any broker system that is architected to use,
// communicate, forward events to the presence system
func NewRedis(server string, db int, inactiveDuration time.Duration) (Backend, error) {
	redis, err := redis.NewRedisSession(&redis.RedisConf{Server: server, DB: db})
	if err != nil {
		return nil, err
	}
	redis.SetPrefix(RedisencePrefix)

	return &Redis{
		redis:                redis,
		becameOfflinePattern: fmt.Sprintf("__keyevent@%d__:expired", db),
		becameOnlinePattern:  fmt.Sprintf("__keyevent@%d__:set", db),
		inactiveDuration:     inactiveDuration,
		errChan:              make(chan error, 1),
	}, nil
}

// Online resets the expiration time for any given key
// if key doesnt exists, it means user is now online and should be set as online
// Whenever application gets any prob from a client
// should call this function
func (s *Redis) Online(ids ...string) error {
	if len(ids) == 1 {
		// if member exits increase ttl
		if s.redis.Expire(ids[0], s.inactiveDuration) == nil {
			return nil
		}

		// if member doesnt exist set it
		return s.redis.Setex(ids[0], s.inactiveDuration, ids[0])
	}

	existance, err := s.sendMultiExpire(ids, s.inactiveDurationString())
	if err != nil {
		return err
	}

	return s.sendMultiSetIfRequired(ids, existance)
}

// Offline sets given ids as offline, ignores any error
func (s *Redis) Offline(ids ...string) error {
	if len(ids) == 1 {
		if s.redis.Expire(ids[0], time.Second*0) == nil {
			return nil
		}
	}

	s.sendMultiExpire(ids, "0")
	return nil
}

// sendMultiSetIfRequired accepts set of ids and their existtance status
// traverse over them and any key is not exists in db, set them in a multi/exec
// request
func (s *Redis) sendMultiSetIfRequired(ids []string, existance []int) error {
	if len(ids) != len(existance) {
		return fmt.Errorf("length is not same Ids: %d Existance: %d", len(ids), len(existance))
	}

	// cache inactive duration
	seconds := s.inactiveDurationString()

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

// sendMultiExpire if the system tries to update more than one key at a time
// inorder to leverage rtt, send multi expire
func (s *Redis) sendMultiExpire(ids []string, duration string) ([]int, error) {
	// get one connection from pool
	c := s.redis.Pool().Get()

	// init multi command
	c.Send("MULTI")

	// send expire command for all members
	for _, id := range ids {
		c.Send("EXPIRE", s.redis.AddPrefix(id), duration)
	}

	// execute command
	r, err := c.Do("EXEC")
	if err != nil {
		return nil, err
	}

	// close connection
	if err := c.Close(); err != nil {
		return nil, err
	}

	values, err := s.redis.Values(r)
	if err != nil {
		return nil, err
	}

	res := make([]int, len(values))
	for i, value := range values {
		res[i], err = s.redis.Int(value)
		if err != nil {
			// what about returning half-generated slice?
			// instead of an empty one
			return nil, err
		}

	}

	return res, nil
}

// Status returns the current status multiple keys from system
func (s *Redis) Status(ids ...string) ([]Event, error) {
	if len(ids) == 1 {
		return s.singleStatus(ids[0])
	}

	// get one connection from pool
	c := s.redis.Pool().Get()

	// init multi command
	c.Send("MULTI")

	// send expire command for all members
	for _, id := range ids {
		c.Send("EXISTS", s.redis.AddPrefix(id))
	}

	// execute command
	r, err := c.Do("EXEC")
	if err != nil {
		return nil, err
	}

	// close connection
	if err := c.Close(); err != nil {
		return nil, err
	}

	values, err := s.redis.Values(r)
	if err != nil {
		return nil, err
	}

	res := make([]Event, len(values))
	for i, value := range values {
		status, err := s.redis.Int(value)
		if err != nil {
			return nil, err
		}

		res[i] = Event{
			ID: ids[i],
			// cast redis response to Status
			Status: Status(status),
		}
	}

	return res, nil
}

// Close closes the redis connection gracefully
func (s *Redis) Close() error {
	return s.close()
}

func (s *Redis) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("closing of already closed connection")
	}

	s.closed = true

	if s.events != nil {
		close(s.events)
	}

	if s.psc != nil {
		s.psc.PUnsubscribe()
	}

	return s.redis.Close()
}

// Status returns the current status a key from system
func (s *Redis) singleStatus(id string) ([]Event, error) {
	res := make([]Event, 1)
	res[0] = Event{
		ID:     id,
		Status: Offline,
	}

	if s.redis.Exists(id) {
		res[0].Status = Online
	}

	return res, nil
}

// ListenStatusChanges subscribes with a pattern to the redis and
// gets online and offline status changes from it
func (s *Redis) ListenStatusChanges() chan Event {
	s.psc = s.redis.CreatePubSubConn()
	s.psc.PSubscribe(s.becameOnlinePattern, s.becameOfflinePattern)

	s.events = make(chan Event)
	go s.listenEvents()
	return s.events
}

// createEvent Creates the event with the required properties
func (s *Redis) listenEvents() {
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		switch n := s.psc.Receive().(type) {
		case gredis.PMessage:
			s.events <- s.createEvent(n)
		case error:
			s.errChan <- n
			return
		}
	}
}

// createEvent Creates the event with the required properties
func (s *Redis) createEvent(n gredis.PMessage) Event {
	e := Event{}

	switch n.Pattern {
	case s.becameOfflinePattern:
		e.ID = string(n.Data[len(RedisencePrefix)+1:])
		e.Status = Offline
	case s.becameOnlinePattern:
		e.ID = string(n.Data[len(RedisencePrefix)+1:])
		e.Status = Online
	default:
		//ignore other events
	}

	return e
}

func (s *Redis) inactiveDurationString() string {
	return strconv.Itoa(int(s.inactiveDuration.Seconds()))
}

// Error returns error if it happens while listening  to status changes
func (s *Redis) Error() chan error {
	return s.errChan
}
