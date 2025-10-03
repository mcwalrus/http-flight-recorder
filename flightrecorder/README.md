# Flight Recorder HTTP

A Go module that provides HTTP endpoints for a registered Go [trace.FlightRecorder](https://pkg.go.dev/golang.org/x/exp/trace#FlightRecorder). This module allows you to easily request snapshots of your application capabilities into your Go applications.

## Features

- **Thread-safe**: All operations are protected by read/write mutexes
- **HTTP Integration**: Easy registration with any `http.ServeMux`
- **Flexible Configuration**: Runtime updates to period and size settings
- **Status Monitoring**: Real-time status of the flight recorder
- **Snapshot Export**: Binary trace data export via HTTP

## Installation

Requires Go versions of 1.25+

```bash
go get flight-recorder/flightrecorder
```

## Usage

### Basic Integration

```go
package main

import (
    "log"
    "net/http"
    
    "flight-recorder/flightrecorder"
)

func main() {
    // Create a new flight recorder service
    flightRecorder := flightrecorder.NewService()

    // Create a new HTTP mux
    mux := http.NewServeMux()

    // Register flight recorder handlers
    flightRecorder.RegisterHandlers(mux)

    // Add your own handlers
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Start the server
    log.Println("Starting server on :8080")
    http.ListenAndServe(":8080", mux)
}
```

### Custom Prefix

```go
// Register with custom prefix
flightRecorder.RegisterHandlersWithPrefix(mux, "/api/v1/debug")
```

### Programmatic Usage

```go
// Create service
service := flightrecorder.NewService()

// Start recording
err := service.Start()

// Get status
status := service.Status()

// Update configuration
updateReq := flightrecorder.UpdateRequest{
    Period: &[]time.Duration{2 * time.Second}[0],
    Size:   &[]int{128 * 1024 * 1024}[0], // 128MB
}
service.Update(updateReq)

// Get snapshot
snapshot, err := service.Snapshot()

// Stop recording
service.Stop()
```

## API Endpoints

### GET /recorder/status
Returns the current status of the flight recorder.

**Response:**
```json
{
  "enabled": false,
  "period": 1000000000,
  "size": 67108864
}
```

### POST /recorder/start
Starts the flight recorder.

### POST /recorder/stop
Stops the flight recorder.

### GET /recorder/snapshot
Returns the current snapshot as binary data.

### POST /recorder/update
Updates the flight recorder configuration.

**Request Body:**
```json
{
  "period": "2s",
  "size": 134217728
}
```

## Configuration

- **Default Period**: 1 second
- **Default Size**: 64MB
- **Thread Safety**: All operations are thread-safe

## Examples

See the `example/` directory for complete usage examples:
- `example/main.go` - Basic integration
- `example/custom_prefix.go` - Custom endpoint prefix
- `example/standalone.go` - Programmatic usage with HTTP server

