package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// LoginStatus tracks a connection's authentication state.
type LoginStatus int

// Login status values.
const (
	LoginFailed LoginStatus = iota
	LoginOK
	LoginClosed
)

// processConn validates a username and registers the connecting player.
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

	zonesMu.Lock()
	_, ok := zones[startingZone]
	zonesMu.Unlock()

	if !ok {
		return LoginFailed, InternalErr
	}

	*player = newPlayer(username, startingZone, conn)

	onlinePlayers[username] = *player

	zones[startingZone].ZoneMu.Lock()
	zones[startingZone].InZone[username] = *player
	zones[startingZone].ZoneMu.Unlock()

	slog.Info("PLAYER_CONNECTED", "remote", conn.RemoteAddr().String(), "player", (*player).Username, "args", args)
	slog.Info(EvtZoneEnter((*player).Username), "loc", startingZone)

	return LoginOK, nil
}

// handleLogin processes CONNECT/QUIT before login completes, with rate limiting.
func handleLogin(conn net.Conn, command, args string, loginState *LoginStatus, player **Player, lastTokenTick *time.Time, chanceCount *float64) {
	const refill = 0.5
	const maxCapacity = 10

	now := time.Now()

	timeElapsed := now.Sub(*lastTokenTick).Seconds()
	tokensToAdd := timeElapsed * refill
	*chanceCount = min(*chanceCount+tokensToAdd, maxCapacity)
	*lastTokenTick = now

	if *chanceCount < 1 {
		_, _ = fmt.Fprintln(conn, InputSpamErr.Error())
		_ = conn.Close()
		*loginState = LoginClosed
		slog.Warn("SYS_MESSAGE", "remote", conn.RemoteAddr().String(), "message", InputSpamErr.Error(), "command", "CONNECT")

		return
	}

	*chanceCount -= 1

	switch command {
	case "CONNECT":
		if args == "" {
			_, _ = fmt.Fprintln(conn, MissingArgsErr.Error())
			slog.Info(MissingArgsErr.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", nil)
			*loginState = LoginFailed

			return
		}

		result, err := processConn(conn, args, player)

		switch err {
		case nil:
			_, _ = fmt.Fprintln(conn, "OK connected")
			slog.Info("SYS_MESSAGE", "player", (*player).Username, "message", "OK connected", "command", command)

		case InternalErr:
			handleInternalError(conn, InternalErr, loginState,
				slog.String("reason", "Invalid starting location value"),
			)

			return

		default:
			_, _ = fmt.Fprintln(conn, err.Error())
			slog.Info(err.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", args)
		}

		*loginState = result

	case "QUIT":
		_, _ = fmt.Fprintln(conn, "OK bye")
		slog.Info("PLAYER_QUIT", "remote", conn.RemoteAddr().String(), "command", command)
		*loginState = LoginClosed
		_ = conn.Close()

	default:
		_, _ = fmt.Fprintln(conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", args)
	}
}

// handleTCPConn drives a single client connection from greeting through cleanup.
func handleTCPConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	_, _ = fmt.Fprintln(conn, "OK hello proto=1")
	slog.Info("SYS_MESSAGE", "remote", conn.RemoteAddr().String(), "message", "OK hello proto=1")

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(time.Second * 30)
	}

	var (
		loginState     LoginStatus
		player         *Player
		lastTokenTick  time.Time
		chanceCount    float64
		deadlineFailed bool
	)

	scantask := bufio.NewScanner(conn)
	const idleTimeout = time.Minute * 5

	for {
		if err := conn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			slog.Warn("SET_READ_DEADLINE_ERROR", "err", err, "remote", conn.RemoteAddr().String())
			deadlineFailed = true
			break
		}

		if !scantask.Scan() {
			break
		}

		line := scantask.Text()
		parts := strings.SplitN(line, " ", 2)

		var args string

		if len(parts) > 1 {
			args = parts[1]
		}

		if loginState == LoginOK {
			commandDispatch(parts[0], args, &loginState, player)

		} else if loginState == LoginClosed {
			break

		} else {
			handleLogin(conn, parts[0], args, &loginState, &player, &lastTokenTick, &chanceCount)
		}
	}

	if deadlineFailed {
		//pass - already logged
	} else if err := scantask.Err(); err != nil {
		var netErr net.Error
		switch {
		case errors.Is(err, net.ErrClosed):
			slog.Info("CONNECTION_CLOSED_SAFELY", "remote", conn.RemoteAddr().String())

		case errors.As(err, &netErr) && netErr.Timeout():
			slog.Warn("CONNECTION_IDLE_TIMEOUT_ERROR", "err", err, "remote", conn.RemoteAddr().String())

		default:
			slog.Warn("CONNECTION_READ_ERROR", "err", err, "remote", conn.RemoteAddr().String())
		}

	} else {
		slog.Info("CONNECTION_CLOSED_BY_CLIENT", "remote", conn.RemoteAddr().String())
	}

	if player != nil {
		player.cleanupPlayerData()
	}
}
