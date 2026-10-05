package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
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

// tapConnection groups everything that belongs to one connection to the server.
type tapConnection struct {
	Conn net.Conn
	ServerResponse chan string
	ServerErr chan error
	Done chan struct{}
	Outgoing chan string
	syncOnce sync.Once
}

// dialServer connects to the server and returns a ready connection, or the dial error.
func dialServer() (*tapConnection, error) {
	const timeOut = time.Second * 10

	conn, err := net.DialTimeout("tcp", ":4242", timeOut)

		if err != nil {
			return nil, fmt.Errorf("could not connect to TCP on 4242: %w", err)
		}

	tc := &tapConnection{
		Conn: conn,
		ServerResponse: make(chan string),
		ServerErr: make(chan error),
		Done: make(chan struct{}),
		Outgoing: make(chan string),
	}

	go tc.readServerLines()

	return tc, nil
}

// readServerLines reads lines until the connection fails and passes each one on.
func (tc *tapConnection) readServerLines() {
	TAPReader := bufio.NewReader(tc.Conn)

	for {
		response, err := TAPReader.ReadString('\n')

		if err != nil {
			select {
			case tc.ServerErr <- err:
			case <- tc.Done:
			}

			return
		}

		select {
		case tc.ServerResponse <- response:
		case <- tc.Done:
			return
		}
		
	}

}

// shutdown stops the goroutines and closes the connection. Safe to call more than once.
func (tc *tapConnection) shutdown() {
	tc.syncOnce.Do(func() {
		_ = tc.Conn.Close()
		close(tc.Done)
	})
}


// main is a temporary test, replaced by the Fyne app later.
func main() {
	tc, err := dialServer()

	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	defer tc.shutdown()

	for {
		select {
		case msg := <- tc.ServerResponse:
			fmt.Print(msg)
		case err := <- tc.ServerErr:
			fmt.Print(err)
			return
		case <-time.After(time.Second * 5):
			return
		}
	}
}




// if strings.HasPrefix(response, "OK") {

// 	}











	
	
	
// 

// serverResponses := make(chan string)
// 	serverErr := make(chan error)

// 	quitTimeoutDuration := time.Second * 3
// 	commandTimeoutDuration := time.Second * 10

// 	go func() {

// 	}()

	
// 					_ = conn.SetWriteDeadline(time.Now().Add(commandTimeoutDuration))
// 				_, err = fmt.Fprintln(conn, input)

				
				
// 								_, _ = fmt.Fprintln(conn, "CONNECT "+input)

								
								
// 				if strings.TrimSpace(response) == "OK bye" {

// 				}
