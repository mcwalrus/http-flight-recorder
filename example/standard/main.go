package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	flightrecorder "flight-recorder"
)

func main() {
	// Create a new flight recorder service
	flightRecorder := flightrecorder.InitService()

	// Create a new HTTP mux
	mux := http.NewServeMux()

	// Register flight recorder handlers with custom prefix
	flightRecorder.RegisterHandlersWithPrefix(mux, "/api/v1/debug/flight")

	// Add your own handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World! Flight recorder is available at /api/v1/debug/flight"))
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("Received signal to stop server")
		flightRecorder.Stop()
		os.Exit(0)
	}()

	// Start the server
	log.Println("Starting server on :8080")
	log.Println("Flight recorder endpoints:")
	log.Println("  GET  /api/v1/debug/flightstatus")
	log.Println("  POST /api/v1/debug/flightstart")
	log.Println("  POST /api/v1/debug/flightstop")
	log.Println("  GET  /api/v1/debug/flightsnapshot")
	log.Println("  POST  /api/v1/debug/flightupdate")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
