package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// generateRandInt returns a random int in [min, max], inclusive.
func generateRandInt(min, max int) int {
	return min + rand.IntN(max-min+1)
}

// getFileData reads a file's full contents, rejecting directories.
func getFileData(pathname string) ([]byte, error) {
	fileInfo, err := os.Stat(pathname)

	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, errors.New("IS_A_DIRECTORY")
	}

	fileData, err := os.ReadFile(pathname)

	if err != nil {
		return nil, err
	}

	return fileData, nil
}

// main loads the world data, starts the TCP listener, and serves connections until shutdown.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	const worldYML = "world.yaml"
	fileData, err := getFileData(worldYML)

	if err != nil {
		slog.Error(InternalErr.Error(), "err", err)
		os.Exit(1)
	}

	err = ParseYmlData(fileData)
	if err != nil {
		slog.Error(InternalErr.Error(), "err", err)
		os.Exit(1)
	}

	errs := ValidateWorldData()
	if len(errs) >= 1 {
		for i, e := range errs {
			slog.Error(InternalErr.Error(), "number", i+1, "err", e)
		}
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":4242")
	if err != nil {
		slog.Error(ConnFailedErr.Error(), "err", err)
		os.Exit(1)
	}

	defer func() { _ = listener.Close() }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Warn("SHUTDOWN_SIGNAL_RECEIVED", "signal", sig.String())
		_ = listener.Close()

		var toCleanUp []*Player
		onlinePlayersMu.Lock()
		onlineCount := len(onlinePlayers)
		for _, p := range onlinePlayers {
			toCleanUp = append(toCleanUp, p)
		}
		slog.Info("PERFORMING_CLEANUP", "players affected", onlineCount)
		onlinePlayersMu.Unlock()

		for _, p := range toCleanUp {
			_, _ = fmt.Fprintln(p.Conn, ConnFailedErr.Error())
			slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", ConnFailedErr.Error())
			_ = p.Conn.Close()
		}
	}()

	ticker := time.NewTicker(time.Minute)

	go func() {
		for {
			select {
			case <-recheckSoftbanCh:
				cleanSoftbanList()

			case <-ticker.C:
				cleanSoftbanList()
			}
		}
	}()

	var currRetries int
	const maxRetries = 5
	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()

		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				slog.Info("SERVER_SHUTTING_DOWN")
				break
			}

			currRetries++
			if currRetries >= maxRetries {
				slog.Error(ConnFailedErr.Error(), "err", err)
				break
			} else {
				slog.Warn(InboundConnErr.Error(), "err", err)
				continue
			}
		} else {
			currRetries = 0
		}

		ipAddress := conn.RemoteAddr().String()
		host, _, _ := net.SplitHostPort(ipAddress)

		trackConnCount(conn, host)

		softBanned, untilWhen := isIPSoftbanned(host, time.Now())
		if softBanned {
			_, _ = fmt.Fprintln(conn, SoftbannedErr.Error())
			slog.Warn(SoftbannedErr.Error(), "host", host, "remote", ipAddress, "until_when", untilWhen, "time_remaining", time.Until(untilWhen))
			_ = conn.Close()
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			handleTCPConn(conn)
		}()
	}

	wg.Wait()
}
