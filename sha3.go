package main

import (
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/sha3"
	"strings"
	"time"
)

func main() {
	var before string
	var after [64]byte
	var base64Encoding string
	var result chan string = make(chan string)
	var resultString string
	var addOnStart int64

	start := time.Now()
	addOn := 0
	concurrency := 8
	sem := make(chan bool, concurrency)

	for {
		before = fmt.Sprintf("Message%d", addOn)
		after = sha3.Sum512([]byte(before))
		base64Encoding = base64.StdEncoding.EncodeToString(after[:])
		if strings.HasPrefix(base64Encoding, "TEST") {
			fmt.Printf("%d seconds\n", int64(time.Since(start)/time.Second))
			break
		}
		addOn++
	}
	fmt.Printf("%d: %s\n", addOn, base64Encoding)

	addOnStart = 0
	start = time.Now()

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

func scan1000000(addOnStart int64, result chan string, sem chan bool) {
	var before string
	var after [64]byte
	var base64Encoding string

	defer func() { <-sem }()

	for i := addOnStart * 1000000; i < (addOnStart+1)*1000000; i++ {
		before = fmt.Sprintf("Message%d", i)
		after = sha3.Sum512([]byte(before))
		base64Encoding = base64.StdEncoding.EncodeToString(after[:])

		if strings.HasPrefix(base64Encoding, "TEST") {
			result <- fmt.Sprintf("%d: %s\n", i, base64Encoding)
			break
		}
	}
}
