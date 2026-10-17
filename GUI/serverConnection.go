package main

import (
	"bufio"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"net"
	"strings"
	"sync"
	"time"
)

// Errors returned when a command can't be sent or QUIT gets no reply.
var (
	ConnClosedErr  = errors.New("sending error: server connection is closed")
	BufferFullErr  = errors.New("sending error: command buffer exceeds limits")
	QuitTimeoutErr = errors.New("no response from server... exiting program anyway")
)

// tapConnection groups everything that belongs to one connection to the server.
type tapConnection struct {
	Conn            net.Conn
	ServerResponses chan string
	ServerErrs      chan error
	Done            chan struct{}
	Outgoing        chan string
	syncOnce        sync.Once
}

// dialServer connects to the server and returns a ready connection, or the dial error.
func dialServer() (*tapConnection, error) {
	const timeOut = time.Second * 10

	conn, err := net.DialTimeout("tcp", ":4242", timeOut)

	if err != nil {
		return nil, fmt.Errorf("could not connect to TCP on 4242: %w", err)
	}

	tc := &tapConnection{
		Conn:            conn,
		ServerResponses: make(chan string),
		ServerErrs:      make(chan error),
		Done:            make(chan struct{}),
		Outgoing:        make(chan string, 16),
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
			case tc.ServerErrs <- err:
			case <-tc.Done:
			}

			return
		}

		select {
		case tc.ServerResponses <- response:
		case <-tc.Done:
			return
		}
	}
}

// writeToServer sends queued commands to the server until the connection fails or is shut down.
func (tc *tapConnection) writeToServer() {
	const commandTimeoutDuration = time.Second * 10

	for {
		select {
		case toSend := <-tc.Outgoing:
			_ = tc.Conn.SetWriteDeadline(time.Now().Add(commandTimeoutDuration))

			_, err := fmt.Fprintln(tc.Conn, toSend)

			if err != nil {
				select {
				case tc.ServerErrs <- err:
				case <-tc.Done:
				}

				return
			}
		case <-tc.Done:
			return
		}
	}
}

// sendButtonCommandToServer queues cmd for sending without blocking. It fails if the
// connection is closed or the queue is full.
func (tc *tapConnection) sendButtonCommandToServer(cmd string) error {
	select {
	case tc.Outgoing <- cmd:
		return nil

	case <-tc.Done:
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

// handleQuit sends QUIT, shows the result on the current screen and closes the window.
func (cr *CommandRouter) handleQuit() {
	if cr.tc != nil {
		var quitMsg string

		quitErr := cr.tc.gracefulQuit(time.Second * 3)
		cr.tc.shutdown()

		if quitErr != nil {
			quitMsg = quitErr.Error()

		} else {
			quitMsg = "OK bye"
		}

		if cr.LoggedIn {
			cr.Game.displayGameError(quitMsg)

		} else {
			fyne.Do(func() {
				cr.Login.ConnErr.SetText(quitMsg)
			})
		}

		time.Sleep(time.Millisecond * 2800)
	}

	fyne.Do(func() {
		cr.Win.Close()
	})
}

// gracefulQuit sends QUIT and waits up to timeout for "OK bye". It returns nil on
// "OK bye", or the connection error or QuitTimeoutErr.
func (tc *tapConnection) gracefulQuit(timeout time.Duration) error {
	if err := tc.sendButtonCommandToServer("QUIT"); err != nil {
		return err
	}

	deadline := time.After(timeout)

	for {
		select {
		case response := <-tc.ServerResponses:
			if strings.TrimSpace(response) == "OK bye" {
				return nil
			}

		case err := <-tc.ServerErrs:
			return err

		case <-tc.Done:
			return ConnClosedErr

		case <-deadline:
			return QuitTimeoutErr
		}
	}
}
