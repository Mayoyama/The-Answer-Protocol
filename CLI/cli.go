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
func gracefulQuit(conn net.Conn, servErr chan error, timeout time.Duration) {
	_, _ = fmt.Fprintln(conn, "QUIT")

	select {
	case err := <-servErr:
		if err != nil {
			_, _ = fmt.Printf("ERROR READING_TO_FROM_SERVER: %v.\nDISCONNECTED.\n", err)
		}

	case <-time.After(timeout):
		fmt.Println("No response from server... exiting program anyway")
	}
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
	greeting, err := TAPReader.ReadString('\n')

	if err != nil {
		fmt.Println("Connected but failed to read from TCP: ", err)
		return
	}

	fmt.Println(greeting)

	var username string

	for username == "" {
		fmt.Println("Input new username: ")

		if !stdinScanner.Scan() {
			fmt.Println("Error reading from stdin: ", stdinScanner.Err())

			return
		}

		input := stdinScanner.Text()
		_, _ = fmt.Fprintln(conn, "CONNECT "+input)

		if response, err := TAPReader.ReadString('\n'); err != nil {
			fmt.Println("Error reading from TAP server: ", err)

			return

		} else {
			if strings.HasPrefix(response, "OK") {
				username = input
			}

			fmt.Println(response)
		}
	}

	serverErr := make(chan error)
	lines := make(chan string)
	stdinErr := make(chan error)

	go func() {
		for {
			response, err := TAPReader.ReadString('\n')
			if err != nil {
				serverErr <- err

				return
			}

			fmt.Print(response)

			if strings.TrimSpace(response) == "OK bye" {
				serverErr <- nil

				return
			}
		}
	}()

	go func() {
		for stdinScanner.Scan() != false {
			lines <- stdinScanner.Text()
		}

		stdinErr <- stdinScanner.Err()
	}()

	for {
		select {
		case input := <-lines:
			_, _ = fmt.Fprintln(conn, input)

		case err := <-serverErr:
			if err != nil {
				_, _ = fmt.Printf("Error reading to/from server: %v.\nDisconnected.\n", err)
			}

			return

		case err := <-stdinErr:
			if err != nil {
				_, _ = fmt.Printf("Error while handling stdin: %v\nClosing connection to server...\n", err)

			} else {
				_, _ = fmt.Println("Ctrl+D/EOF detected. Closing connection to server...")
			}

			gracefulQuit(conn, serverErr, 3*time.Second)

			return

		case <-sigCh:
			_, _ = fmt.Println("Ctrl+C or shutdown request detected. Closing connection to server...")

			gracefulQuit(conn, serverErr, 3*time.Second)

			return
		}
	}
}
