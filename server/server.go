package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	yml "gopkg.in/yaml.v3"
)

type LoginStatus int

const (
	LoginFailed LoginStatus = iota
	LoginOK
	LoginClosed
)

func processConn(conn net.Conn, args string, player **Player) (LoginStatus, error) {
	username := strings.TrimSpace(args)

	if strings.ContainsFunc(username, func(r rune) bool {
		return !unicode.IsPrint(r)
	}) {
		return LoginFailed, InvCharInNameErr

	} else if utf8.RuneCountInString(username) > 10 {
		return LoginFailed, NameTooLongErr

	} else if utf8.RuneCountInString(username) < 2 {
		return LoginFailed, NameTooShortErr
	}

	onlinePlayersMu.Lock()
	defer onlinePlayersMu.Unlock()

	_, isOnline := onlinePlayers[username]

	if isOnline {
		return LoginFailed, NameInUseErr
	}

	var startingZone string = "taverne"

	zonesMu.Lock()
	_, ok := zones[startingZone]
	zonesMu.Unlock()

	if !ok {
		return LoginFailed, InternalErr
	}

	*player = NewPlayer(username, startingZone, conn)

	onlinePlayers[username] = *player

	zones[startingZone].ZoneMu.Lock()
	zones[startingZone].InZone[username] = *player
	zones[startingZone].ZoneMu.Unlock()

	slog.Info("PLAYER_CONNECTED", "player", (*player).Username, "args", args)
	slog.Info(EvtZoneEnter((*player).Username), "zone", startingZone)

	return LoginOK, nil
}

func handleLogin(conn net.Conn, command, args string, loginState *LoginStatus, player **Player) {
	command = strings.ToUpper((command))

	switch command {
	case "CONNECT":
		if args == "" {
			fmt.Fprintln(conn, MissingArgsErr.Error())
			slog.Info(MissingArgsErr.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", nil)
			*loginState = LoginFailed

			return
		}

		result, err := processConn(conn, args, player)

		if err != nil {
			fmt.Fprintln(conn, err.Error())
			slog.Info(err.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", args)

		} else {
			fmt.Fprintln(conn, "OK connected")
			slog.Info("SYS_MESSAGE", "player", (*player).Username, "message", "OK connected", "command", command)
		}

		*loginState = result

	case "QUIT":
		fmt.Fprintln(conn, "OK bye")
		slog.Info("PLAYER_QUIT", "remote", conn.RemoteAddr().String(), "command", command)
		*loginState = LoginClosed
		conn.Close()

	default:
		fmt.Fprintln(conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", args)
	}
}

func handleTCPConn(conn net.Conn) {
	defer conn.Close()
	fmt.Fprintln(conn, "OK hello proto=1")

	var loginState LoginStatus
	var player *Player

	scantask := bufio.NewScanner(conn)
	for scantask.Scan() {
		line := scantask.Text()
		parts := strings.SplitN(line, " ", 2)

		var args string

		if len(parts) > 1 {
			args = parts[1]
		}

		if loginState == LoginOK {
			commandDispatch(conn, parts[0], args, &loginState, player)

		} else if loginState == LoginClosed {
			break

		} else {
			handleLogin(conn, parts[0], args, &loginState, &player)
		}
	}

	if err := scantask.Err(); err != nil {
		slog.Warn("CONNECTION_READ_ERROR", "err", err, "remote", conn.RemoteAddr().String())
	}

	if player != nil {
		player.CleanupPlayerData()
	}
}

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

func main() {
	worldYML := "world.yaml"
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

	err = ValidateWorldData()
	if err != nil {
		slog.Error(InternalErr.Error(), "err", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":4242")
	if err != nil {
		slog.Error(ConnFailedErr.Error(), "err", err)
		os.Exit(1)
	}
	defer listener.Close()

	var currRetries int
	maxRetries := 5
	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()
		if err != nil {
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

		wg.Add(1)
		go func() {
			defer wg.Done()
			handleTCPConn(conn)
		}()
	}

	wg.Wait()
}
