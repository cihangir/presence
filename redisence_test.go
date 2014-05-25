package redisence

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func initRedisence(t *testing.T) *Session {
	ses, err := New("localhost:6379", 10, time.Second*1)
	if err != nil {
		t.Fatal(err)
	}

	return ses
}

func TestInitialization(t *testing.T) {
	initRedisence(t)
}

func TestPing(t *testing.T) {
	if err := initRedisence(t).Ping("id"); err != nil {
		t.Fatal(err)
	}
}

func TestStatus(t *testing.T) {
	s := initRedisence(t)
	if err := s.Ping("id"); err != nil {
		t.Fatal(err)
	}
	if err := s.Ping("id"); err != nil {
		t.Fatal(err)
	}
	status := s.Status("id")
	if status == Offline {
		t.Fatal(errors.New("User should be active"))
	}
}

func TestSubscriptions(t *testing.T) {
	s := initRedisence(t)

	events := make(chan Event, 10)

	go s.ListenStatusChanges(events)

	for event := range events {
		switch event.Status {
		case Online:
			fmt.Println(event)
		case Offline:
			fmt.Println(event)
		case Closed:
			close(events)
			fmt.Println(event)
			return
		}
	}
}

func TestStatusWithTimeout(t *testing.T) {
	if err := initRedisence(t).Ping("id"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 1)
	status := initRedisence(t).Status("id")
	if status == Online {
		t.Fatal(errors.New("User should be active"))
	}
}
