package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
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

var (
	netConns     = make(map[net.Conn]struct{})
	netConnsMu   sync.Mutex
	shuttingDown bool
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
		startZoneErr := fmt.Errorf("%w: INVALID_START_ZONE %s", InternalErr, startingZone)
		return LoginFailed, startZoneErr
	}

	newP := newPlayer(username, startingZone, conn)

	if newP.MaxHP <= 0 || newP.CurrHP != newP.MaxHP {
		hpErr := fmt.Errorf("%w: INVALID_HP_VALUE(S) %d/%d", InternalErr, newP.CurrHP, newP.MaxHP)
		return LoginFailed, hpErr
	}

	_, writeErr := fmt.Fprintln(conn, "OK connected")

	if writeErr != nil {
		connErr := fmt.Errorf("%w: OK_CONNECTED_WRITE_FAILURE %v", InternalErr, writeErr)

		return LoginFailed, connErr
	}

	slog.Info("SYS_MESSAGE", "player", username, "message", "OK connected", "command", "CONNECT")

	*player = newP

	onlinePlayers[username] = *player

	onPlayerCount := len(onlinePlayers)

	_, _ = fmt.Fprintln(conn, EvtPlayerCount(onPlayerCount))
	slog.Info("SYS_MESSAGE", "recipient", username, "message", EvtPlayerCount(onPlayerCount), "reason", "player joined server")

	zones[startingZone].ZoneMu.Lock()

	for k, p := range zones[startingZone].InZone {
		_, _ = fmt.Fprintln(p.Conn, EvtZoneEnter(newP.Username))
		slog.Info(EvtZoneEnter(newP.Username), "recipient", k, "loc", startingZone)
	}

	zones[startingZone].InZone[username] = *player
	zones[startingZone].ZoneMu.Unlock()

	slog.Info("PLAYER_CONNECTED", "remote", conn.RemoteAddr().String(), "player", username, "args", args)
	slog.Info(EvtZoneEnter(username), "loc", startingZone)

	for k, p := range onlinePlayers {
		if k != username {
			_, _ = fmt.Fprintln(p.Conn, EvtPlayerCount(len(onlinePlayers)))
			slog.Info("SYS_MESSAGE", "recipient", k, "message", EvtPlayerCount(len(onlinePlayers)), "reason", "player joined server")
		}
	}

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

		if errors.Is(err, InternalErr) {
			handleInternalError(conn, err, loginState,
				slog.String("remote", conn.RemoteAddr().String()),
			)

			return
		}

		if err != nil {
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
	defer func() {
		_ = conn.Close()
		netConnsMu.Lock()
		delete(netConns, conn)
		netConnsMu.Unlock()
	}()

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
	const idleTimeout = time.Minute * 15

	for {
		if loginState == LoginClosed {
			break
		}

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
			commandDispatch(strings.ToUpper(parts[0]), args, &loginState, player)

		} else if loginState == LoginClosed {
			break

		} else {
			handleLogin(conn, strings.ToUpper(parts[0]), args, &loginState, &player, &lastTokenTick, &chanceCount)
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
