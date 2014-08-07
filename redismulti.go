package presence

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	redigo "github.com/garyburd/redigo/redis"
	"github.com/koding/redis"
)

const (
	faultTolerantPresencePrefix = "ftpp"
	OnlineKey                   = "online"
)

// Redis holds the required connection data for redis
type FaultTolerantRedis struct {
	// main redis connection
	redis *redis.RedisSession

	// inactiveDuration specifies no-probe allowance time
	inactiveDuration time.Duration

	// errChan pipe all errors  the this channel
	errChan chan error

	// closed holds the status of connection
	closed bool

	// notfiy closing
	closeChan chan struct{}

	// holds event channel
	events chan Event

	// lock for Redis struct
	mu sync.Mutex
}

func NewFaultTolerantRedis(server string, db int, inactiveDuration time.Duration) (Backend, error) {
	// TODO connect to all redis instances
	redis, err := redis.NewRedisSession(&redis.RedisConf{Server: server, DB: db})
	if err != nil {
		return nil, err
	}
	redis.SetPrefix(PresencePrefix)
	s := &FaultTolerantRedis{
		redis:            redis,
		inactiveDuration: inactiveDuration,
		errChan:          make(chan error, 1),
		closeChan:        make(chan struct{}, 1),
		events:           make(chan Event, 1000),
	}

	go s.StarOfflineWatcher(time.Millisecond * 100)

	return s, nil
}

// Online sets given ids as online in the system
func (f *FaultTolerantRedis) Online(ids ...string) error {
	t := getTime()

	// TODO first check if users are currently online
	events, err := f.Status(ids...)
	if err != nil {
		return err
	}

	err = addUsersToConnection(f.redis, t, ids...)
	if err != nil {
		return err
	}

	newOnlines := make([]string, 0)
	for _, event := range events {
		if event.Status == Offline {
			newOnlines = append(newOnlines, event.Id)
		}
	}

	if len(newOnlines) != 0 {
		// no need to block
		go f.sendEvents(Online, t, newOnlines...)
	}

	return nil
}

// Offline sets given ids as offline, ignores any error
func (f *FaultTolerantRedis) Offline(ids ...string) error {
	t := getTime()

	err := removeUsersFromConnection(f.redis, t, ids...)
	if err != nil {
		return err
	}

	// no need to block for sending events
	go f.sendEvents(Offline, t, ids...)

	return nil
}

// Status returns the current status multiple keys from system
func (f *FaultTolerantRedis) Status(ids ...string) ([]Event, error) {
	return getStatusFromConnection(f.redis, ids...)
}

func (f *FaultTolerantRedis) ListenStatusChanges() chan Event {
	return f.events
}

// Error returns error if it happens while listening  to status changes
func (f *FaultTolerantRedis) Error() error {
	return <-f.errChan
}

func addUsersToConnection(redis *redis.RedisSession, seenAt string, ids ...string) error {
	_, err := redis.AddSetMembers(OnlineKey, sliceStringToSliceInterface(ids)...)
	if err != nil {
		return err
	}

	err = setLatestStatus(redis, seenAt, "1", ids...)
	if err != nil {
		return err
	}

	return nil
}

func removeUsersFromConnection(redis *redis.RedisSession, seenAt string, ids ...string) error {
	_, err := redis.RemoveSetMembers(OnlineKey, sliceStringToSliceInterface(ids)...)
	if err != nil {
		return err
	}

	err = setLatestStatus(redis, seenAt, "0", ids...)
	if err != nil {
		return err
	}

	return nil
}

func setLatestStatus(redis *redis.RedisSession, seenAt string, status string, ids ...string) error {

	// get one connection from pool
	c := redis.Pool().Get()
	c.Send("MULTI")

	for _, id := range ids {
		c.Send("HMSET", redis.AddPrefix(id), "status", status, "seenAt", seenAt)
	}

	// ignore values
	if _, err := c.Do("EXEC"); err != nil {
		return err
	}

	// do not forget to close the connection
	if err := c.Close(); err != nil {
		return err
	}

	return nil
}

func getStatusFromConnection(redis *redis.RedisSession, ids ...string) ([]Event, error) {
	// get one connection from pool
	c := redis.Pool().Get()

	// init multi command
	c.Send("MULTI")

	// send expire command for all members
	for _, id := range ids {
		c.Send("HGETALL", redis.AddPrefix(id))
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

	values, err := redis.Values(r)
	if err != nil {
		return nil, err
	}

	res := make([]Event, len(values))
	for i, value := range values {
		val, err := redis.Values(value)
		if err != nil {
			return nil, err
		}

		if err := redigo.ScanStruct(val, &res[i]); err != nil {
			return nil, err
		}

		// todo be sure that, we have equal number of response with ids
		res[i].Id = ids[i]
	}

	return res, nil
}

// Close closes all the redis connections gracefully
func (f *FaultTolerantRedis) Close() error {
	return f.close()
}

func (f *FaultTolerantRedis) close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return errors.New("Closing of already closed connection")
	}

	f.closed = true

	// close closeChan
	close(f.closeChan)

	if f.events != nil {
		close(f.events)
		f.events = nil
	}

	// TODO - close all connnections
	return f.redis.Close()
}

func sliceStringToSliceInterface(items []string) []interface{} {
	d := make([]interface{}, len(items))
	for i, item := range items {
		d[i] = item
	}

	return d
}

func getTime() string {
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}

func (f *FaultTolerantRedis) StarOfflineWatcher(gcInterval time.Duration) {
	go func() {
		// TODO optimize this ticker according to slow consumers
		for _ = range time.Tick(gcInterval) {

			if f.closed == true {
				return
			}

			currentTime := time.Now().UTC().UnixNano()

			card, err := f.redis.Scard(OnlineKey)
			if err != nil {
				continue
			}

			for i := 0; i < card; i++ {
				fmt.Println("willl run with this iter i-->", i)
				// having a random key will cover the presence server crash, we will
				// start working on the obtained key, will have the global lock, if
				// server crashes, because of the TTL on global lock will be
				// released and another presence server can get the same key and
				// process it
				//
				// TODO, when we have multiple redis instance, get random redis
				// connection
				key, err := f.redis.PopSetMember(OnlineKey)
				if err == redis.ErrNil {
					// if no item found in the online set, wait for next tick
					break
				}

				//
				// TODO obtain a global lock for `key` if we can not get the global
				// lock, continue working on another key
				//

				// get the locked key's info from multiple servers
				val, err := f.redis.HashGetAll(key)
				if err != nil {
					f.errChan <- err
					continue
				}

				res := &Event{}
				if err := redigo.ScanStruct(val, res); err != nil {
					f.errChan <- err
					continue
				}

				// if status is now online, check user's last seen at date
				//  mark user as -offline- that means user
				// was online(because it was in "online" set), but we marked it as
				// offline between last check and current check
				if res.Status == Online {
					if res.SeenAt < currentTime-int64(f.inactiveDuration) {
						// set user id
						res.Id = key
						res.Status = Offline
						// send user as Offline event
						f.events <- *res

						// remove user from online set
						f.redis.RemoveSetMembers(OnlineKey, key)
					} else {
						f.redis.AddSetMembers(OnlineKey, key)
					}

					//
					// TODO - release the global lock
					//
				}
			}

		}
	}()
}

// doesnt block on sending events, just drops messages if no one is listening
func (f *FaultTolerantRedis) sendEvents(event Status, t string, ids ...string) error {
	seenAt, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return err
	}

	for _, id := range ids {
		e := Event{
			Id:     id,
			Status: event,
			SeenAt: seenAt,
		}
		select {
		case f.events <- e:
		case <-f.closeChan:
			break
		default:
			// buffer overflow
			// drop message
			continue
		}

	}

	return nil
}
