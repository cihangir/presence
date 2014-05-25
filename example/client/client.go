package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/siesta/redisence"
)

func main() {
	// if os.Getenv("GOMAXPROCS") == "" {
	// 	runtime.GOMAXPROCS(runtime.NumCPU())
	// }

	session, err := redisence.New("localhost:6379", 10, time.Second*1)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		pingdom(session, 1, 3000)
		wg.Done()
	}()
	go func() {
		pingdom(session, 3001, 6000)
		wg.Done()
	}()

	go func() {
		pingdom(session, 6001, 9000)
		wg.Done()
	}()

	// go func() {
	// 	for i := 0; i < 3333; i++ {
	// 		session.Status(strconv.Itoa(i))
	// 	}
	// 	wg.Done()
	// }()
	wg.Done()

	wg.Wait()
}

func pingdom(session *redisence.Session, start, end int) {
	throttleCount := 1000
	req := make([]string, throttleCount)
	count := 0
	for i := start; i <= end; i++ {
		req[count] = strconv.Itoa(i)
		if count == throttleCount-1 {
			session.Ping(req...)
			count = 0
		}
		count++
	}
}
