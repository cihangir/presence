// Package main show a simple client implementation
package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/cihangir/presence"
)

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	backend, err := presence.NewRedis("localhost:6379", 10, time.Second*1)
	if err != nil {
		panic(err)
	}

	session, err := presence.New(backend)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		pingdom(session, 1, 30000)
		wg.Done()
	}()

	// request data from database
	go func() {
		statuko(session, 1, 30000)
		wg.Done()
	}()

	wg.Wait()
}

func pingdom(session *presence.Session, start, end int) {
	throttleCount := 1500
	req := make([]string, throttleCount)
	count := 0
	for i := start; i <= end; i++ {
		req[count] = strconv.Itoa(i)
		if count == throttleCount-1 {
			err := session.Online(req...)
			if err != nil {
				fmt.Println(err)
			}

			count = 0
		}
		count++
	}
}

func statuko(session *presence.Session, start, end int) {
	throttleCount := 1500
	req := make([]string, throttleCount)
	count := 0
	for i := start; i <= end; i++ {
		req[count] = strconv.Itoa(i)
		if count == throttleCount-1 {
			_, err := session.Status(req...)
			if err != nil {
				fmt.Println(err)
			}

			count = 0
		}
		count++
	}
}
