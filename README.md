# gosha3
Golang script to test sha3 operations in serial, parallel, and distributed client-server modes

## Features
- **Local Mode**: Single-threaded or multi-threaded hash searching
- **Server Mode**: Distribute work across multiple clients
- **Client Mode**: Connect to server and perform distributed hash computation
- **Automatic Coordination**: Server automatically exits when result is found
- **Connection Retry**: Clients wait for server availability

## Installation
```bash
go get golang.org/x/crypto/sha3 # first time
```

## Usage

### Local Mode (Original)
```bash
# Single-threaded
go run sha3.go -search="TEST"

# Multi-threaded (4 goroutines)
go run sha3.go -concurrency=4 -search="TEST"
```

### Client-Server Mode (Distributed)

#### 1. Start the Server
```bash
# Start server on port 8080
go run sha3.go -server=8080 -search="TEST"
```

The server will:
- Display startup information
- Wait for client connections
- Distribute work in 1-million item chunks
- Automatically exit when a result is found
- Show total runtime and which client found the result

#### 2. Start Clients
```bash
# Connect to server with 2 workers
go run sha3.go -client=localhost:8080 -concurrency=2 -search="TEST"

# Connect to remote server
go run sha3.go -client=192.168.1.100:8080 -concurrency=4 -search="TEST"
```

Clients will:
- Wait up to 30 seconds for server availability
- Request work ranges from server
- Submit results back to server
- Display local computation time

#### 3. Multiple Clients Example
```bash
# Terminal 1: Start server
go run sha3.go -server=8080 -search="TEST"

# Terminal 2: Start first client
go run sha3.go -client=localhost:8080 -concurrency=2 -search="TEST"

# Terminal 3: Start second client
go run sha3.go -client=localhost:8080 -concurrency=4 -search="TEST"

# Terminal 4: Start third client
go run sha3.go -client=localhost:8080 -concurrency=1 -search="TEST"
```

## Command Line Options

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `-search` | String to search for in hash prefix | "TEST" | `-search="ABC"` |
| `-concurrency` | Number of goroutines (local/client mode) | 1 | `-concurrency=4` |
| `-server` | Server mode: listen on port | - | `-server=8080` |
| `-client` | Client mode: connect to server | - | `-client=localhost:8080` |

## Server Output Example
```
Server starting on port 8080
Search string: TEST
Endpoints:
  POST /work - Request work range
  POST /found - Submit found result
Waiting for clients...
=== RESULT FOUND ===
Found by client: client-1
Total server runtime: 3 seconds
Result: 13135013: TESTLhbwflN1qKfkNzkV7I4l55mCzyn7kKNyX1VArpTNaxfMssDrrg6dzQTPgN6h7Mhc+0qUi9PFH3YWBn5SGg==
===================
Server shutting down...
```

## Client Output Example
```
Connecting to server at localhost:8080...
Connected to server successfully!
2 seconds
13135013: TESTLhbwflN1qKfkNzkV7I4l55mCzyn7kKNyX1VArpTNaxfMssDrrg6dzQTPgN6h7Mhc+0qUi9PFH3YWBn5SGg==
```

## API Endpoints

### Server Endpoints

#### POST /work
Request work from server
```json
{
  "client_id": "client-0"
}
```

Response:
```json
{
  "start_range": 0,
  "end_range": 1000000,
  "search_string": "TEST",
  "found": false
}
```

#### POST /found
Submit found result to server
```json
{
  "work_item": 13135013,
  "base64_encoding": "TESTLhbwflN1qKfkNzkV7I4l55mCzyn7kKNyX1VArpTNaxfMssDrrg6dzQTPgN6h7Mhc+0qUi9PFH3YWBn5SGg==",
  "client_id": "client-0"
}
```

## Troubleshooting

### Server Issues
- **Port already in use**: Change port number or kill existing process
- **Permission denied**: Use a port > 1024 or run with sudo

### Client Issues
- **Connection refused**: Ensure server is running and accessible
- **Timeout**: Client waits 30 seconds for server; check network connectivity

### Performance Tips
- Use multiple clients with different concurrency levels
- Distribute clients across multiple machines for better performance
- Monitor server output to see which client finds results first
