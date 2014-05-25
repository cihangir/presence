package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/siesta/redisence"
)

func main() {
	session, err := redisence.New("localhost:6379", 10, time.Second*1)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for i := 0; i < 10000; i++ {
			session.Ping(strconv.Itoa(i))
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 10000; i++ {
			session.Status(strconv.Itoa(i))
		}
		wg.Done()
	}()
	wg.Wait()
}
