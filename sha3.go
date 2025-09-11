package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/sha3"
)

var testString string

// WorkRequest represents a request for work from client
type WorkRequest struct {
	ClientID string `json:"client_id"`
}

// WorkResponse represents work assignment from server
type WorkResponse struct {
	StartRange int64  `json:"start_range"`
	EndRange   int64  `json:"end_range"`
	SearchStr  string `json:"search_string"`
	Found      bool   `json:"found"`
}

// FoundRequest represents a found result from client
type FoundRequest struct {
	WorkItem       int64  `json:"work_item"`
	Base64Encoding string `json:"base64_encoding"`
	ClientID       string `json:"client_id"`
}

// Server state
var (
	serverCounter int64
	serverFound   int32
	serverResult  string
	serverMutex   sync.RWMutex
	serverStart   time.Time
)

func main() {
	var before string
	var after [64]byte
	var base64Encoding string
	var result chan string = make(chan string)
	var resultString string
	var concurrencyFlag = flag.Int("concurrency", 1, "Number of goroutines to run simultaneously")
	var testStringFlag = flag.String("search", "TEST", "String to search for")
	var clientFlag = flag.String("client", "", "Client mode: connect to server at ipaddr:port")
	var serverFlag = flag.String("server", "", "Server mode: listen on port")
	var start time.Time

	flag.Parse()
	concurrency := *concurrencyFlag
	testString = *testStringFlag

	// Handle server mode
	if *serverFlag != "" {
		startServer(*serverFlag)
		return
	}

	// Handle client mode
	if *clientFlag != "" {
		startClient(*clientFlag, concurrency)
		return
	}

	// Original local mode
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

// startServer starts the HTTP server for work distribution
func startServer(port string) {
	serverStart = time.Now()
	http.HandleFunc("/work", handleWorkRequest)
	http.HandleFunc("/found", handleFoundResult)

	fmt.Printf("Server starting on port %s\n", port)
	fmt.Printf("Search string: %s\n", testString)
	fmt.Println("Endpoints:")
	fmt.Println("  POST /work - Request work range")
	fmt.Println("  POST /found - Submit found result")
	fmt.Println("Waiting for clients...")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// handleWorkRequest handles work requests from clients
func handleWorkRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if already found
	if atomic.LoadInt32(&serverFound) == 1 {
		response := WorkResponse{Found: true}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Assign work range (1 million items per request)
	startRange := atomic.AddInt64(&serverCounter, 1000000) - 1000000
	endRange := startRange + 1000000

	response := WorkResponse{
		StartRange: startRange,
		EndRange:   endRange,
		SearchStr:  testString,
		Found:      false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFoundResult handles found results from clients
func handleFoundResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if already found by another client
	if atomic.CompareAndSwapInt32(&serverFound, 0, 1) {
		serverMutex.Lock()
		serverResult = fmt.Sprintf("%d: %s\n", req.WorkItem, req.Base64Encoding)
		serverMutex.Unlock()

		totalTime := time.Since(serverStart)
		fmt.Printf("\n=== RESULT FOUND ===\n")
		fmt.Printf("Found by client: %s\n", req.ClientID)
		fmt.Printf("Total server runtime: %d seconds\n", int64(totalTime/time.Second))
		fmt.Printf("Result: %s", serverResult)
		fmt.Printf("===================\n")

		// Exit the server after a short delay to allow response to be sent
		go func() {
			time.Sleep(100 * time.Millisecond)
			fmt.Println("Server shutting down...")
			os.Exit(0)
		}()
	}

	w.WriteHeader(http.StatusOK)
}

// startClient starts the client to request work from server
func startClient(serverAddr string, concurrency int) {
	fmt.Printf("Connecting to server at %s...\n", serverAddr)

	// Wait for server to be available
	if !waitForServer(serverAddr) {
		fmt.Println("Failed to connect to server after multiple attempts")
		return
	}

	fmt.Println("Connected to server successfully!")
	start := time.Now()

	// Start worker goroutines
	var wg sync.WaitGroup
	var found int32
	result := make(chan string, 1)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(clientID string) {
			defer wg.Done()
			clientWorker(serverAddr, clientID, result, &found)
		}(fmt.Sprintf("client-%d", i))
	}

	// Wait for result
	resultString := <-result
	atomic.StoreInt32(&found, 1)

	// Wait for all workers to finish
	wg.Wait()

	fmt.Printf("%d seconds\n", int64(time.Since(start)/time.Second))
	fmt.Print(resultString)
}

// clientWorker is a worker that requests work from server
func clientWorker(serverAddr, clientID string, result chan string, found *int32) {
	for {
		// Check if result was already found
		if atomic.LoadInt32(found) == 1 {
			return
		}

		// Request work from server
		workRange, err := requestWork(serverAddr, clientID)
		if err != nil {
			fmt.Printf("Error requesting work: %v\n", err)
			time.Sleep(time.Second) // Wait before retrying
			continue
		}

		// If server says we're done, exit
		if workRange.Found {
			return
		}

		// Do the work
		if foundResult := doWork(workRange.StartRange, workRange.EndRange, workRange.SearchStr, found); foundResult != "" {
			// Submit result to server
			submitResult(serverAddr, foundResult, clientID)

			// Try to send to local result channel
			select {
			case result <- foundResult:
			default:
			}
			return
		}
	}
}

// requestWork requests work from the server
func requestWork(serverAddr, clientID string) (*WorkResponse, error) {
	req := WorkRequest{ClientID: clientID}
	jsonData, _ := json.Marshal(req)

	resp, err := http.Post("http://"+serverAddr+"/work", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var workResp WorkResponse
	if err := json.NewDecoder(resp.Body).Decode(&workResp); err != nil {
		return nil, err
	}

	return &workResp, nil
}

// doWork performs the actual hash computation work
func doWork(startRange, endRange int64, searchStr string, found *int32) string {
	var before string
	var after [64]byte
	var base64Encoding string

	for i := startRange; i < endRange; i++ {
		// Check if result was already found
		if atomic.LoadInt32(found) == 1 {
			return ""
		}

		before = fmt.Sprintf("Message%d", i)
		after = sha3.Sum512([]byte(before))
		base64Encoding = base64.StdEncoding.EncodeToString(after[:])

		if strings.HasPrefix(base64Encoding, searchStr) {
			return fmt.Sprintf("%d: %s\n", i, base64Encoding)
		}
	}

	return ""
}

// submitResult submits a found result to the server
func submitResult(serverAddr, result, clientID string) {
	// Parse the result to extract work item and base64 encoding
	parts := strings.Split(result, ": ")
	if len(parts) != 2 {
		return
	}

	workItem, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}

	base64Encoding := strings.TrimSpace(parts[1])

	req := FoundRequest{
		WorkItem:       workItem,
		Base64Encoding: base64Encoding,
		ClientID:       clientID,
	}

	jsonData, _ := json.Marshal(req)
	http.Post("http://"+serverAddr+"/found", "application/json", strings.NewReader(string(jsonData)))
}

// waitForServer waits for the server to become available
func waitForServer(serverAddr string) bool {
	maxRetries := 30 // 30 seconds total
	retryDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		// Try to make a simple HTTP request to check if server is up
		resp, err := http.Get("http://" + serverAddr + "/work")
		if err == nil {
			resp.Body.Close()
			return true
		}

		if i < maxRetries-1 {
			fmt.Printf("Server not available, retrying in %v... (%d/%d)\n", retryDelay, i+1, maxRetries)
			time.Sleep(retryDelay)
		}
	}

	return false
}
