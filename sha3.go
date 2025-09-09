package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/sha3"
)

var testString string

func main() {
	var before string
	var after [64]byte
	var base64Encoding string
	var result chan string = make(chan string)
	var resultString string
	var concurrencyFlag = flag.Int("concurrency", 1, "Number of goroutines to run simultaneously")
	var testStringFlag = flag.String("search", "TEST", "String to search for")
	var start time.Time

	flag.Parse()
	concurrency := *concurrencyFlag
	testString = *testStringFlag
	if concurrency == 1 {
		addOn := 0
		start = time.Now()

		for {
			before = fmt.Sprintf("Message%d", addOn)
			after = sha3.Sum512([]byte(before))
			base64Encoding = base64.StdEncoding.EncodeToString(after[:])
			if strings.HasPrefix(base64Encoding, testString) {
				fmt.Printf("%d seconds\n", int64(time.Since(start)/time.Second))
				break
			}
			addOn++
		}
		fmt.Printf("%d: %s\n", addOn, base64Encoding)

	} else {
		start = time.Now()

		// Use atomic counter for thread-safe incrementing
		var counter int64
		var found int32 // atomic flag to signal when result is found
		var wg sync.WaitGroup

		// Start worker goroutines
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				scan1000000Optimized(&counter, result, &found)
			}()
		}

		// Wait for result
		resultString = <-result
		atomic.StoreInt32(&found, 1) // Signal all workers to stop

		// Wait for all workers to finish
		wg.Wait()

		fmt.Printf("%d seconds\n", int64(time.Since(start)/time.Second))
		fmt.Print(resultString)
	}
}

func scan1000000(addOnStart int64, result chan string, sem chan bool) {
	var before string
	var after [64]byte
	var base64Encoding string

	defer func() { <-sem }()

	for i := addOnStart * 1000000; i < (addOnStart+1)*1000000; i++ {
		before = fmt.Sprintf("Message%d", i)
		after = sha3.Sum512([]byte(before))
		base64Encoding = base64.StdEncoding.EncodeToString(after[:])

		if strings.HasPrefix(base64Encoding, testString) {
			result <- fmt.Sprintf("%d: %s\n", i, base64Encoding)
			break
		}
	}
}

// Optimized version that uses atomic operations and early termination
func scan1000000Optimized(counter *int64, result chan string, found *int32) {
	var before string
	var after [64]byte
	var base64Encoding string

	for {
		// Check if result was already found by another goroutine
		if atomic.LoadInt32(found) == 1 {
			return
		}

		// Atomically get next work item
		workItem := atomic.AddInt64(counter, 1) - 1

		before = fmt.Sprintf("Message%d", workItem)
		after = sha3.Sum512([]byte(before))
		base64Encoding = base64.StdEncoding.EncodeToString(after[:])

		if strings.HasPrefix(base64Encoding, testString) {
			// Try to send result, but don't block if channel is full
			select {
			case result <- fmt.Sprintf("%d: %s\n", workItem, base64Encoding):
				return
			default:
				// Another goroutine already found a result
				return
			}
		}
	}
}
