package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/siesta/redisence"
)

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	session, err := redisence.New("localhost:6379", 10, time.Second*1)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		pingdom(session, 1, 30000)
		wg.Done()
	}()
	go func() {
		pingdom(session, 30001, 60000)
		wg.Done()
	}()

	go func() {
		pingdom(session, 60001, 90000)
		wg.Done()
	}()

	go func() {
		statuko(session, 1, 30000)
		wg.Done()
	}()

	wg.Wait()
}

func pingdom(session *redisence.Session, start, end int) {
	throttleCount := 1500
	req := make([]string, throttleCount)
	count := 0
	for i := start; i <= end; i++ {
		req[count] = strconv.Itoa(i)
		if count == throttleCount-1 {
			err := session.Ping(req...)
			if err != nil {
				fmt.Println(err)
			}

			count = 0
		}
		count++
	}
}

func statuko(session *redisence.Session, start, end int) {
	throttleCount := 1500
	req := make([]string, throttleCount)
	count := 0
	for i := start; i <= end; i++ {
		req[count] = strconv.Itoa(i)
		if count == throttleCount-1 {
			err := session.Status(req)
			if err != nil {
				fmt.Println(err)
			}

			count = 0
		}
		count++
	}
}
