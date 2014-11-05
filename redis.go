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

var (
	// Prefix for presence package
	Prefix = "presence"

	// ErrInvalidID for stating the event id is not valid
	ErrInvalidID = errors.New("invalid id")

	// ErrInvalidStatus for stating the event status is not valid
	ErrInvalidStatus = errors.New("invalid status")
)

// Redis holds the required connection data for redis
type Redis struct {
	// main redis connection
	redis *redis.RedisSession

	// inactiveDuration specifies no-probe allowance time
	inactiveDuration string

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

// NewRedis creates a Redis presence system
func NewRedis(
	server string, // connection string
	db int, // redis db number
	inactiveDuration time.Duration, // timeout duration
) (Backend, error) {

	// create the redis connection
	redis, err := redis.NewRedisSession(&redis.RedisConf{
		Server: server,
		DB:     db,
	})
	if err != nil {
		return nil, err
	}

	// for prefix for redis backend
	redis.SetPrefix(Prefix)

	return &Redis{
		redis:                redis,
		becameOfflinePattern: fmt.Sprintf("__keyevent@%d__:expired", db),
		becameOnlinePattern:  fmt.Sprintf("__keyevent@%d__:set", db),
		inactiveDuration:     strconv.Itoa(int(inactiveDuration.Seconds())),
		errChan:              make(chan error, 1),
	}, nil
}

// Online resets the expiration time for any given key. If key doesnt exists, it
// means key (user) become online and should be set as online. Whenever
// application gets any probe from a client should call this function. This
// method performs way better when there is a throttling mechanism implemented
// on top of it, please refer to benchmarks
func (s *Redis) Online(ids ...string) error {
	// try to send expire command in a batch request. `Expire` command will
	// reply with integer reply - 0 or 1 for a given key -
	// http://redis.io/topics/protocol#integer-reply. If the response is 0 that
	// means the key doesnt exist in our system. You can read more about redis
	// `Exist` command here http://redis.io/commands/exists
	existance, err := s.multiExpire(ids, s.inactiveDuration)
	if err == nil {
		return s.multiSetIfRequired(ids, existance, Error{})
	}

	// if err is not a multi err, return it
	e, ok := err.(Error)
	if !ok {
		return err
	}

	// we have a multierr here, so process following operations with that info
	if err := s.multiSetIfRequired(ids, existance, e); err != nil {
		return err
	}

	return e
}

// Offline sets given ids as offline
func (s *Redis) Offline(ids ...string) error {
	const zeroTimeString = "0"
	_, err := s.multiExpire(ids, zeroTimeString)
	return err
}

// Status returns the current status of multiple keys from system
func (s *Redis) Status(ids ...string) ([]Event, error) {
	// get one connection from pool
	c := s.redis.Pool().Get()
	// close connection
	defer c.Close()

	// init multi command
	c.Send("MULTI")

	// send exists command for all members
	for _, id := range ids {
		c.Send("EXISTS", s.redis.AddPrefix(id))
	}

	// execute command
	r, err := c.Do("EXEC")
	if err != nil {
		return nil, err
	}

	values, err := s.redis.Values(r)
	if err != nil {
		return nil, err
	}

	e := Error{}
	res := make([]Event, len(values))
	for i, value := range values {
		status, err := s.redis.Int(value)
		if err != nil {
			e.Append(ids[i], err)
			continue
		}

		res[i] = Event{
			ID: ids[i],
			// cast redis response to Status
			Status: redisResToStatus[status],
		}
	}

	if e.Len() > 0 {
		// generally returning a meaningful data along with an error is not a
		// good idea, but here, most of the calls may success, we should not
		// stop the execution on first error
		return res, e
	}

	return res, nil
}

// Error returns error if it happens while listening  to status changes
func (s *Redis) Error() chan error {
	return s.errChan
}

// Close closes the redis connection gracefully
func (s *Redis) Close() error {
	return s.close()
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

var redisResToStatus = map[int]Status{
	// redis response for exists command is 0 when the id is not in the system
	0: Offline,

	// redis response for exists command is 1 when the id is in the system
	1: Online,
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

	// if incoming data len is smaller than our prefix, do not process the event
	if len(n.Data) < len(Prefix) {
		s.errChan <- ErrInvalidID
		return e
	}

	e.ID = string(n.Data[len(Prefix)+1:])

	switch n.Pattern {
	case s.becameOfflinePattern:
		e.Status = Offline
	case s.becameOnlinePattern:
		e.Status = Online
	default:
		s.errChan <- ErrInvalidStatus
	}

	return e
}

// multiSetIfRequired accepts a set of ids and their existance status
func (s *Redis) multiSetIfRequired(ids []string, existance []int, e Error) error {
	// redis ensures that all the responses in a transaction will be in the same
	// order with the requests. So we can safely assume that our keys and their
	// responses are in the same order. For more info
	// http://redis.io/topics/transactions
	if len(ids) != len(existance) {
		return fmt.Errorf("length is not same Ids: %d Existance: %d", len(ids), len(existance))
	}

	// get one connection from pool
	c := s.redis.Pool().Get()
	// do not forget to close the connection
	defer c.Close()

	// item count for non-existent members
	notExistsCount := 0

	// traverse over all the given keys and any key is not exists in db, set
	// them in a multi/exec request
	for i, exists := range existance {
		// `0` means, member does not exists in presence system
		if exists != 0 {
			continue
		}

		// if we got any error for the current id, do not process it
		if e.Has(ids[i]) {
			continue
		}

		// init multi command lazily
		if notExistsCount == 0 {
			if err := c.Send("MULTI"); err != nil {
				return err
			}
		}

		notExistsCount++

		// if we reach to that point, set the new key into system
		err := c.Send("SETEX", s.redis.AddPrefix(ids[i]), s.inactiveDuration, ids[i])
		if err != nil {
			e.Append(ids[i], err)
		}
	}

	// execute multi command if only we flushed some to connection
	if notExistsCount != 0 {
		// ignore values
		if _, err := c.Do("EXEC"); err != nil {
			return err
		}
	}

	// if we have any multierr, return it back
	if e.Len() > 0 {
		return e
	}

	return nil
}

// multiExpire if the system tries to update more than one key at a time
// inorder to leverage rtt, send multi expire
func (s *Redis) multiExpire(ids []string, duration string) ([]int, error) {
	// get one connection from pool
	c := s.redis.Pool().Get()
	// close connection
	defer c.Close()

	// init multi command
	if err := c.Send("MULTI"); err != nil {
		return nil, err
	}

	e := Error{}

	// send expire command for all members
	for _, id := range ids {
		err := c.Send("EXPIRE", s.redis.AddPrefix(id), duration)
		if err != nil {
			e.Append(id, err)
		}
	}

	// execute command
	r, err := c.Do("EXEC")
	if err != nil {
		return nil, err
	}

	values, err := s.redis.Values(r)
	if err != nil {
		return nil, err
	}

	e := Error{}
	res := make([]int, len(values))
	for i, value := range values {
		res[i], err = s.redis.Int(value)
		if err != nil {
			e.Append(ids[i], err)
		}

	}

	// if we got any error, return them all along with result set
	if e.Len() > 0 {
		return res, e
	}

	return res, nil
}
