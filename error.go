package presence

import (
	"bytes"
	"fmt"
)

// Error holds the non-thread safe errors for given specific ids
type Error map[string]error

// Append adds an error to the aggregated errors with an id
func (m *Error) Append(id string, err error) {
	(*m)[id] = err
}

// Has checks if the Error has an error for the given id
func (m *Error) Has(id string) bool {
	_, has := (*m)[id]
	return has
}

// Len returns the registered error count
func (m *Error) Len() int {
	return len(*m)
}

// Each iterates over error set with calling the given function
func (m *Error) Each(f func(id string, err error)) {
	for id, err := range *m {
		f(id, err)
	}
}

// Error implements the error interface
func (m Error) Error() string {
	buf := &bytes.Buffer{}
	buf.WriteString("Presence Error:")

	m.Each(func(id string, err error) {
		fmt.Fprintf(buf, "{id: %s, err:%s}", id, err.Error())
	})

	return buf.String()
}
