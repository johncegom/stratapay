package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	ctx := context.Background()
	const key = "lock:two-step-demo"

	const n = 20
	var wg sync.WaitGroup
	var wins int64

	for i := range n {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ok, err := rdb.SetNX(ctx, key, id, 500*time.Millisecond).Result()
			if err != nil {
				fmt.Printf("goroutine %d: error: %v\n", id, err)
				return
			}
			if ok {
				atomic.AddInt64(&wins, 1)
				fmt.Printf("goroutine %d: ACQUIRED the lock\n", id)
			} else {
				fmt.Printf("goroutine %d: lock already held\n", id)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("\ntotal winners: %d (expected: 1)\n", wins)

}
