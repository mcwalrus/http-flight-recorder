package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	flightrecorder "flight-recorder"
)

const (
	serverAddr = "localhost:8083"
	baseURL    = "http://" + serverAddr
)

type FlightRecorderCLI struct {
	server   *http.Server
	client   *http.Client
	baseURL  string
	shutdown chan os.Signal
}

func NewFlightRecorderCLI() *FlightRecorderCLI {
	return &FlightRecorderCLI{
		client:   &http.Client{Timeout: 5 * time.Second},
		baseURL:  baseURL,
		shutdown: make(chan os.Signal, 1),
	}
}

func (cli *FlightRecorderCLI) StartServer() {
	flightRecorder := flightrecorder.InitService()

	mux := http.NewServeMux()
	flightRecorder.RegisterHandlers(mux)

	cli.server = &http.Server{
		Addr:    ":" + strings.Split(serverAddr, ":")[1],
		Handler: mux,
	}

	go func() {
		<-cli.shutdown
		log.Println("Received signal to stop server")
		flightRecorder.Stop()
		cli.server.Close()
		os.Exit(0)
	}()

	go func() {
		log.Printf("Starting server on %s\n", serverAddr)
		log.Println("Flight recorder endpoints:")
		log.Println("  GET  /recorder/status")
		log.Println("  POST /recorder/start")
		log.Println("  POST /recorder/stop")
		log.Println("  GET  /recorder/snapshot")
		log.Println("  POST  /recorder/update")
		log.Println("")

		if err := cli.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	time.Sleep(100 * time.Millisecond) // short wait
}

func (cli *FlightRecorderCLI) StopServer() {
	if cli.server != nil {
		cli.server.Close()
	}
}

func (cli *FlightRecorderCLI) GetStatus() error {
	resp, err := cli.client.Get(cli.baseURL + "/recorder/status")
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", string(body))
	}

	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("failed to parse status: %w", err)
	}

	fmt.Printf("Flight Recorder Status:\n")
	fmt.Printf("  Enabled: %t\n", status["enabled"])
	fmt.Printf("  Period: %v\n", status["period"])
	fmt.Printf("  Size: %d bytes\n", status["size"])
	return nil
}

func (cli *FlightRecorderCLI) StartFlightRecorder() error {
	resp, err := cli.client.Post(cli.baseURL+"/recorder/start", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to start flight recorder: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp flightrecorder.ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("server error: %s", errorResp.Error)
		}
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Println("Flight recorder started successfully!")
	return nil
}

func (cli *FlightRecorderCLI) StopFlightRecorder() error {
	resp, err := cli.client.Post(cli.baseURL+"/recorder/stop", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to stop flight recorder: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp flightrecorder.ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("server error: %s", errorResp.Error)
		}
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Println("Flight recorder stopped successfully!")
	return nil
}

func (cli *FlightRecorderCLI) GetSnapshot() error {
	resp, err := cli.client.Get(cli.baseURL + "/recorder/snapshot")
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp flightrecorder.ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("server error: %s", errorResp.Error)
		}
		return fmt.Errorf("server error: %s", string(body))
	}

	// Save snapshot to file
	filename := fmt.Sprintf("snapshot_%d.trace", time.Now().Unix())
	if err := os.WriteFile(filename, body, 0644); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	fmt.Printf("Snapshot saved to %s (%d bytes)\n", filename, len(body))
	return nil
}

func (cli *FlightRecorderCLI) UpdateFlightRecorder() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter new period in seconds (press Enter to skip): ")
	periodStr, _ := reader.ReadString('\n')
	periodStr = strings.TrimSpace(periodStr)

	fmt.Print("Enter new size in MB (press Enter to skip): ")
	sizeStr, _ := reader.ReadString('\n')
	sizeStr = strings.TrimSpace(sizeStr)

	updateReq := flightrecorder.UpdateRequest{}

	if periodStr != "" {
		if period, err := strconv.Atoi(periodStr); err == nil {
			periodDuration := time.Duration(period) * time.Second
			updateReq.Period = &periodDuration
		} else {
			return fmt.Errorf("invalid period: %s", periodStr)
		}
	}

	if sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			sizeBytes := size * 1024 * 1024 // Convert MB to bytes
			updateReq.Size = &sizeBytes
		} else {
			return fmt.Errorf("invalid size: %s", sizeStr)
		}
	}

	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := cli.client.Post(cli.baseURL+"/recorder/update", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update flight recorder: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp flightrecorder.ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("server error: %s", errorResp.Error)
		}
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Println("Flight recorder configuration updated successfully!")
	return nil
}

func (cli *FlightRecorderCLI) PrintHelp() {
	fmt.Println("\n=== Flight Recorder CLI ===")
	fmt.Println("Available commands:")
	fmt.Println("  s - Get status")
	fmt.Println("  1 - Start flight recorder")
	fmt.Println("  2 - Stop flight recorder")
	fmt.Println("  3 - Get snapshot")
	fmt.Println("  4 - Update configuration")
	fmt.Println("  h - Show this help")
	fmt.Println("  q - Quit")
	fmt.Println("")
}

func (cli *FlightRecorderCLI) Run() {
	signal.Notify(cli.shutdown, os.Interrupt, syscall.SIGTERM)

	cli.StartServer()
	cli.PrintHelp()

	reader := bufio.NewReader(os.Stdin)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		command := strings.TrimSpace(strings.ToLower(input))

		switch command {
		case "s", "status":
			if err := cli.GetStatus(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "1", "start":
			if err := cli.StartFlightRecorder(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "2", "stop":
			if err := cli.StopFlightRecorder(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "3", "snapshot":
			if err := cli.GetSnapshot(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "4", "update":
			if err := cli.UpdateFlightRecorder(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "h", "help":
			cli.PrintHelp()
		case "q", "quit", "exit":
			fmt.Println("Shutting down...")
			cli.StopServer()
			return
		case "":
			// Empty input, continue
			continue
		default:
			fmt.Printf("Unknown command: %s. Type 'h' for help.\n", command)
		}

		fmt.Println() // Add blank line after each command
	}
}

func main() {
	cli := NewFlightRecorderCLI()
	cli.Run()
}
