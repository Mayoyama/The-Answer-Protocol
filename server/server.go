package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
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

	var startingZone string = "J'SAIS PAS"

	*player = NewPlayer(username, startingZone, conn)
	onlinePlayers[username] = *player

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
		slog.Info("PLAYER_QUIT", "player", (*player).Username, "command", command)
		*loginState = LoginClosed
		conn.Close()

	default:
		fmt.Fprintln(conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "remote", conn.RemoteAddr().String(), "command", command, "args", args)
	}
}

func commandDispatch(conn net.Conn, command, args string, loginState *LoginStatus, player *Player) {
	switch command {
	case "CONNECT":
		fmt.Fprintln(conn, AlreadyConnErr.Error())
		slog.Info(AlreadyConnErr.Error(), "player", player.Username, "command", command, "args", args)

	case "LOOK", "QUIT", "WHO", "STATUS", "INVENTORY", "QUESTS":
		if args != "" {
			fmt.Fprintln(conn, InvalidArgsErr.Error())
			slog.Info(InvalidArgsErr.Error(), "player", player.Username, "command", command, "args", nil)

		} else {
			if command == "QUIT" {
				fmt.Fprintln(conn, "OK bye")
				slog.Info("PLAYER_QUIT", "player", player.Username, "command", command)
				*loginState = LoginClosed
				conn.Close()

			} else if command == "LOOK" {
				//function here

			} else if command == "WHO" {
				onlinePlayersMu.Lock()
				defer onlinePlayersMu.Unlock()
				fmt.Fprintf(conn, "OK players=%d\n", len(onlinePlayers))
				slog.Info("SYS_MESSAGE", "player", player.Username, "message", "OK players="+strconv.Itoa(len(onlinePlayers)), "command", command)

			} else if command == "STATUS" {
				player.PlayerMu.Lock()
				defer player.PlayerMu.Unlock()
				playerStats := PlayerStatusResponse{
					HP:     player.CurrHP,
					MaxHP:  player.MaxHP,
					Status: player.Status,
				}
				statPrint, err := json.Marshal(playerStats)

				if err != nil {
					fmt.Fprintln(conn, JSONErr.Error())
					slog.Error(JSONErr.Error(), "player", player.Username, "command", command, "args", args)
					return
				}

				fmt.Fprintln(conn, "OK", string(statPrint))
				slog.Info("SYS_MESSAGE", "player", player.Username, "message", "OK "+string(statPrint), "command", command)

			} else if command == "INVENTORY" {
				player.PlayerMu.Lock()
				defer player.PlayerMu.Unlock()

				bag := make([]string, 0, len(player.Inventory))
				for item := range player.Inventory {
					bag = append(bag, item)
				}

				sac, err := json.Marshal(bag)

				if err != nil {
					fmt.Fprintln(conn, JSONErr.Error())
					slog.Error(JSONErr.Error(), "player", player.Username, "command", command, "args", args)
					return
				}

				fmt.Fprintln(conn, "OK", string(sac))
				slog.Info("SYS_MESSAGE", "player", player.Username, "message", "OK "+string(sac), "command", command)

			} else if command == "QUESTS" {
				//function here
			}
		}
	case "MOVE", "CHAT", "GROUP", "TAKE", "DROP", "TALK", "ATTACK", "QUEST":
		if args == "" {
			//TODO: Need to comment about custom error in readme
			fmt.Fprintln(conn, MissingArgsErr.Error())
			slog.Info(MissingArgsErr.Error(), "player", player.Username, "command", command, "args", args)

		} else {
			subparts := strings.SplitN(args, " ", 2)
			var subargs string
			if len(subparts) > 1 {
				subargs = subparts[1]
			}

			if command == "MOVE" {
				//function here

			} else if command == "CHAT" {
				chatDispatcher(subparts[0], subargs, player)

			} else if command == "GROUP" {
				groupFuncDispatcher(subparts[0], subargs, player)

			} else if command == "TAKE" {
				//function here

			} else if command == "DROP" {
				//function here

			} else if command == "TALK" {
				//function here

			} else if command == "ATTACK" {
				//function here

			} else if command == "QUEST" {
				//function here
			}
		}

	default:
		fmt.Fprintln(conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "player", player.Username, "command", command, "args", args)
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
}

func main() {
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
