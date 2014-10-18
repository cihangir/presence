// Package presence provides an advanced presence system
package presence

const (
	// Offline is for displaying user as offline in the system
	Offline Status = iota + 1 // do not handle unset variable as offline

	// Online is for displaying user as online in the system
	Online
)

const (
	online  = "ONLINE"
	offline = "OFFLINE"
	unknown = "UNKNOWN"
)

// Status defines what is the current status of a user in presence system
type Status int

// String implements the stringer interface
func (s Status) String() string {
	switch s {
	case Offline:
		return offline
	case Online:
		return online

	}

	// if status is not set or not a known value
	return unknown
}

// Backend represents basic interface for all required backend operations for
// presence package
type Backend interface {
	Online(...string) error
	Offline(...string) error
	Status(...string) ([]Event, error)
	Close() error
	Error() chan error
	ListenStatusChanges() chan Event
}

// Event is the data type for occuring events in the system
type Event struct {
	// ID is the given key by the application
	ID string

	// Status holds the changing type of event
	Status Status
}

// Session holds the backend and provides accessor methods for communication
type Session struct {
	// holds the interface
	backend Backend
}

// New creates a session for any broker system that is architected to use,
// communicate, forward events to the presence system
func New(backend Backend) (*Session, error) {
	return &Session{backend: backend}, nil
}

// Online sets given ids as online
func (s *Session) Online(ids ...string) error {
	return s.backend.Online(ids...)
}

// Offline sets given ids as offline
func (s *Session) Offline(ids ...string) error {
	return s.backend.Offline(ids...)
}

// Status returns the current status of multiple keys from system
func (s *Session) Status(ids ...string) ([]Event, error) {
	return s.backend.Status(ids...)
}

// Close closes the backend connection gracefully
func (s *Session) Close() error {
	return s.backend.Close()
}

// Error returns error if it happens while listening  to status changes
func (s *Session) Error() chan error {
	return s.backend.Error()
}

// ListenStatusChanges subscribes the backend and gets online and offline status
// changes from it
func (s *Session) ListenStatusChanges() chan Event {
	return s.backend.ListenStatusChanges()
}
