package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// gracefulQuit sends QUIT to the server and waits briefly for a response before exiting.
func gracefulQuit(conn net.Conn, serverResponses chan string, serverErr chan error, timeout time.Duration) {
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	_, _ = fmt.Fprintln(conn, "QUIT")

	deadline := time.After(timeout)

	for {
		select {
		case response := <-serverResponses:
			if strings.TrimSpace(response) == "OK bye" {
				_, _ = fmt.Print(response)

				return
			}

		case err := <-serverErr:
			if err != nil {
				_, _ = fmt.Printf("ERROR READING_TO_FROM_SERVER: %v.\nDISCONNECTED.\n", err)

				return
			}

		case <-deadline:
			fmt.Println("No response from server... exiting program anyway")

			return
		}
	}
}

// handleLoginReply prints the server's reply to CONNECT and returns the accepted
// username, or "" if the reply wasn't OK (in which case it prompts again).
func handleLoginReply(response, input string) string {
	var username string

	if strings.HasPrefix(response, "OK") {
		username = input
	}

	_, _ = fmt.Println(response)

	if username == "" {
		_, _ = fmt.Println("Enter a new username between 2 and 10 characters:")
	}

	return username
}

// main connects to the server, logs the player in, and relays stdin/server traffic until disconnect.
func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	conn, err := net.Dial("tcp", ":4242")

	if err != nil {
		fmt.Println("Could not connect to TCP on 4242: ", err)

		return
	}

	defer func() { _ = conn.Close() }()

	TAPReader := bufio.NewReader(conn)
	stdinScanner := bufio.NewScanner(os.Stdin)

	serverResponses := make(chan string)
	serverErr := make(chan error)
	userInput := make(chan string)
	stdinErr := make(chan error)

	quitTimeoutDuration := time.Second * 3
	commandTimeoutDuration := time.Second * 10

	go func() {
		for {
			response, err := TAPReader.ReadString('\n')
			if err != nil {
				serverErr <- err

				return
			}

			serverResponses <- response
		}
	}()

	go func() {
		for stdinScanner.Scan() {
			userInput <- stdinScanner.Text()
		}

		stdinErr <- stdinScanner.Err()
	}()

	var (
		username     string
		nameInput    string
		TCPConnected bool
		// stdinGateLine blocks userInput channel when nil. It stays nil until the server greeting
		// has been printed, and while a CONNECT reply is pending, so only one CONNECT is ever active.
		stdinGateLine chan string
	)

	for {
		select {
		case input := <-stdinGateLine:
			if username == "" {
				nameInput = input
				stdinGateLine = nil

				_, _ = fmt.Fprintln(conn, "CONNECT "+input)

			} else {
				_ = conn.SetWriteDeadline(time.Now().Add(commandTimeoutDuration))
				_, err = fmt.Fprintln(conn, input)

				if err != nil {
					_, _ = fmt.Printf("Error reading to/from server: %v.\nDisconnected.\n", err)

					return
				}
			}

		case response := <-serverResponses:
			switch {
			case !TCPConnected:
				TCPConnected = true

				_, _ = fmt.Println(response)
				_, _ = fmt.Println("Enter a new username between 2 and 10 characters:")

				stdinGateLine = userInput

			case username == "":
				username = handleLoginReply(response, nameInput)
				stdinGateLine = userInput

			default:
				_, _ = fmt.Print(response)

				if strings.TrimSpace(response) == "OK bye" {
					return
				}
			}

		case err := <-serverErr:
			switch {
			case !TCPConnected:
				_, _ = fmt.Println("Connected but failed to read from TCP server: ", err)

			case username == "":
				_, _ = fmt.Println("Error reading from TAP server: ", err)

			default:
				_, _ = fmt.Printf("Error reading to/from server: %v.\nDisconnected.\n", err)
			}

			return

		case err := <-stdinErr:
			if err != nil {
				_, _ = fmt.Printf("Error while handling stdin: %v\nClosing connection to server...\n", err)

			} else {
				_, _ = fmt.Println("Ctrl+D/EOF detected. Closing connection to server...")
			}

			gracefulQuit(conn, serverResponses, serverErr, quitTimeoutDuration)

			return

		case <-sigCh:
			_, _ = fmt.Println("Ctrl+C or shutdown request detected. Closing connection to server...")

			gracefulQuit(conn, serverResponses, serverErr, quitTimeoutDuration)

			return
		}
	}
}
