// Package redisence provides simple user presence system
package redisence

// Status defines what is the current status of a user
// in presence system
type Status int

const (
	Offline Status = iota
	Online
)

type Backend interface {
	Online(...string) error
	Offline(...string) error
	Status(...string) ([]Event, error)
	Close() error
	Error() error
	ListenStatusChanges() chan Event
}

// Event is the data type for
// occuring events in the system
type Event struct {
	// Id is the given key by the application
	Id string

	// Status holds the changing type of event
	Status Status
}

type Session struct {
	// holds the interface
	backend Backend
}

// New creates a session for any broker system that is architected to use,
// communicate, forward events to the presence system
func New(backend Backend) (*Session, error) {
	return &Session{backend: backend}, nil
}

// Online sets given ids as online, ignores any error
func (s *Session) Online(ids ...string) error {
	return s.backend.Online(ids...)
}

// Offline sets given ids as offline, ignores any error
func (s *Session) Offline(ids ...string) error {
	return s.backend.Offline(ids...)
}

// MultipleStatus returns the current status multiple keys from system
func (s *Session) Status(ids ...string) ([]Event, error) {
	return s.backend.Status(ids...)
}

// Close closes the redis connection gracefully
func (s *Session) Close() error {
	return s.backend.Close()
}

// Error returns error if it happens while listening  to status changes
func (s *Session) Error() error {
	return s.backend.Error()
}

// ListenStatusChanges subscribes with a pattern to the redis and
// gets online and offline status changes from it
func (s *Session) ListenStatusChanges() chan Event {
	return s.backend.ListenStatusChanges()
}
