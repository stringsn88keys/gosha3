package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"golang.org/x/crypto/sha3"
	"strings"
	"time"
)

var testString string

func main() {
	var before string
	var after [64]byte
	var base64Encoding string
	var result chan string = make(chan string)
	var resultString string
	var addOnStart int64
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
		addOnStart = 0
		start = time.Now()

		sem := make(chan bool, concurrency)
	SEARCHY:
		for {
			sem <- true
			go scan1000000(addOnStart, result, sem)

			select {
			case resultString, _ = <-result:
				break SEARCHY
			default:
			}
			addOnStart++
		}

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
