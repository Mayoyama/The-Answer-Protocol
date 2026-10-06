package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

var (
	ConnClosedErr = errors.New("sending error: server connection is closed")
	BufferFullErr = errors.New("sending error: command buffer exceeds limits")
	QuitTimeoutErr = errors.New("no response from server... exiting program anyway")
)

// gracefulQuit sends QUIT to the server and waits briefly for a response before exiting.
func (tc *tapConnection) gracefulQuit(timeout time.Duration) error {
	if err := tc.sendButtonCommandToServer("QUIT"); err != nil {
		return err
	}

	deadline := time.After(timeout)

	for {
		select {
		case response := <- tc.ServerResponse:
			if strings.TrimSpace(response) == "OK bye" {
				return nil
			}

		case err := <-tc.ServerErr:
			return err
			
		case <- tc.Done:
			return ConnClosedErr

		case <-deadline:
			return QuitTimeoutErr
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
		Outgoing: make(chan string, 16),
	}

	go tc.readServerLines()
	go tc.writeToServer()

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

func (tc *tapConnection) writeToServer() {
	const commandTimeoutDuration = time.Second * 10
	
	for {
		select {
		case toSend := <- tc.Outgoing:
			_ = tc.Conn.SetWriteDeadline(time.Now().Add(commandTimeoutDuration))
			
			_, err := fmt.Fprintln(tc.Conn, toSend)
			
			if err != nil {
				select {
				case tc.ServerErr <- err:
				case <- tc.Done:
				}

				return
			}
		case <- tc.Done:
			return
		}
	}
}

func (tc *tapConnection) sendButtonCommandToServer(cmd string) error {
	select {
	case tc.Outgoing <- cmd:
		return nil

	case <- tc.Done:
		return ConnClosedErr

	default:
		return BufferFullErr
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
	
	const quitTimeoutDuration = time.Second * 3

	for {
		select {
		case response := <- tc.ServerResponse:
			if strings.HasPrefix(response, "OK hello proto") {
				if err := tc.sendButtonCommandToServer("CONNECT input"); err != nil {
					_, _ = fmt.Println(err)

				return
				}
			}

			fmt.Print(response)

		case err := <- tc.ServerErr:
			_, _ = fmt.Println(err)

			return

		case <-time.After(time.Second * 5):
			if err := tc.gracefulQuit(quitTimeoutDuration); err != nil {
				_, _ = fmt.Println(err)

			} else {
				_, _ = fmt.Println("OK bye")
			}
			return
		}
	}
}

// 				if strings.TrimSpace(response) == "OK bye" {

// 				}
