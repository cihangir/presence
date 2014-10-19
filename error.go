package presence

import (
	"bytes"
	"fmt"
)

// PresenceError holds the errors for given specific ids
type PresenceError map[string]error

// Append adds an error to the aggregated errors with an id
func (m *PresenceError) Append(id string, err error) {
	(*m)[id] = err
}

// Has checks if the PresenceError has an error for the given id
func (m *PresenceError) Has(id string) bool {
	_, has := (*m)[id]
	return has
}

// Len returns the registered error count
func (m *PresenceError) Len() int {
	return len(*m)
}

// Each iterates over error set with calling the given function
func (m *PresenceError) Each(f func(id string, err error)) {
	for id, err := range *m {
		f(id, err)
	}
}

// Error implements the error interface
func (m PresenceError) Error() string {
	buf := &bytes.Buffer{}
	buf.WriteString("Presence Error:")

	m.Each(func(id string, err error) {
		fmt.Fprintf(buf, "{id: %s, err:%s}", id, err.Error())
	})

	return buf.String()
}
