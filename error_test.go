package presence

import (
	"errors"
	"fmt"
	"testing"
)

	id1 := <-nextId
	id2 := <-nextId
func TestErrorLen(t *testing.T) {
	e := Error{}
	err1 := errors.New(id1)
	err2 := errors.New(id2)

	e.Append(id1, err1)
	e.Append(id2, err2)

	if e.Len() != 2 {
		t.Fatalf("err len should be 2, but got:", e.Len())
	}
}

	id := <-nextId
func TestErrorEach(t *testing.T) {
	e := Error{}
	err := errors.New(id)

	e.Append(id, err)
	e.Each(func(idx string, errx error) {
		if idx != id {
			t.Fatalf("error id should be: %s, but got: %s", id, idx)
		}
		if err.Error() != errx.Error() {
			t.Fatalf("error message should be: %s, but got: %s", err.Error(), errx.Error())
		}
	})
}

	id1 := <-nextId
	id2 := <-nextId
func TestErrorString(t *testing.T) {
	e := Error{}
	err1 := errors.New(id1)
	err2 := errors.New(id2)

	e.Append(id1, err1)
	e.Append(id2, err2)

	errMessage := fmt.Sprintf("Presence Error:{id: %[1]s, err:%[1]s}{id: %[2]s, err:%[2]s}", id1, id2)
	if e.Error() != errMessage {
		t.Fatalf("err message should be \"%s\", but got: \"%s\"", errMessage, e.Error())
	}
}

	id1 := <-nextId
	id2 := <-nextId
func TestErrorHas(t *testing.T) {
	e := Error{}
	err1 := errors.New(id1)

	e.Append(id1, err1)

	if !e.Has(id1) {
		t.Fatalf("multi err should have %s, but it doesnt", id1)
	}
	if e.Has(id2) {
		t.Fatalf("multi err should not have %s, but it does", id2)
	}
}
