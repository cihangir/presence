// Package redisence provides simple user presence system
package redisence

import (
	"fmt"
	"strconv"
	"time"

	gredis "github.com/garyburd/redigo/redis"
	"github.com/koding/redis"
)

// Status defines what is the current status of a user
// in presence system
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

	// inactiveDurationAsString holds the expiration duration as string
	inactiveDurationAsString string

	// closeChan is used for giving close signal
	closeChan chan bool

	// errChan pipe all errors  the this channel
	errChan chan error

	// closed holds the status of connection
	closed bool

	//psc holds the pubsub channel if opened
	psc *gredis.PubSubConn
}

// New creates a session for any broker system that is architected to use,
// communicate, forward events to the presence system
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
		// cache inactive duration as string
		inactiveDurationAsString: strconv.Itoa(int(inactiveDuration.Seconds())),
		closeChan:                make(chan bool, 1),
		errChan:                  make(chan error, 1),
	}, nil
}

// Close closes the redis connection gracefully
func (s *Session) Close() error {
	s.closeChan <- true
	return s.close()
}

// Error returns error if it happens while listening  to status changes
func (s *Session) Error() error {
	return <-s.errChan
}

// Ping resets the expiration time for any given key
// if key doesnt exists, it means user is now online and should be set as online
// Whenever application gets any prob from a client
// should call this function
func (s *Session) Online(ids ...string) error {
	if len(ids) == 1 {
		// if member exits increase ttl
		if s.redis.Expire(ids[0], s.inactiveDuration) == nil {
			return nil
		}

		// if member doesnt exist set it
		return s.redis.Setex(ids[0], s.inactiveDuration, ids[0])
	}

	existance, err := s.sendMultiExpire(ids, s.inactiveDurationAsString)
	if err != nil {
		return err
	}

	return s.sendMultiSetIfRequired(ids, existance)
}

// Offline sets given ids as offline, ignores any error
// since not exist keys returned as nilErr
func (s *Session) Offline(ids ...string) error {
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

// sendMultiExpire if the system tries to update more than one key at a time
// inorder to leverage rtt, send multi expire
func (s *Session) sendMultiExpire(ids []string, duration string) ([]int, error) {
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

// MultipleStatus returns the current status multiple keys from system
func (s *Session) MultipleStatus(ids []string) ([]Event, error) {
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
			Id: ids[i],
			// cast redis response to Status
			Status: Status(status),
		}
	}

	return res, nil
}

// Status returns the current status a key from system
func (s *Session) Status(id string) (Event, error) {
	res := Event{
		Id:     id,
		Status: Offline,
	}

	if s.redis.Exists(id) {
		res.Status = Online
	}

	return res, nil
}

// ListenStatusChanges subscribes with a pattern to the redis and
// gets online and offline status changes from it
func (s *Session) ListenStatusChanges(events chan Event) {
	s.psc = s.redis.CreatePubSubConn()
	s.psc.PSubscribe(s.becameOnlinePattern, s.becameOfflinePattern)

	go s.listenEvents(events)

	<-s.closeChan
	events <- Event{Status: Closed}
}

func (s *Session) close() error {
	s.closed = true
	if s.psc != nil {
		s.psc.PUnsubscribe()
	}

	return s.redis.Close()
}

// createEvent Creates the event with the required properties
func (s *Session) listenEvents(events chan Event) {
	for {
		if s.closed {
			break
		}
		switch n := s.psc.Receive().(type) {
		case gredis.PMessage:
			events <- s.createEvent(n)
		case error:
			s.errChan <- n
			return
		}
	}
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
