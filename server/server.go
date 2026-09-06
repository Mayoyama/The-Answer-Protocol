package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"unicode"
)

type LoginStatus int

const (
	LoginFailed LoginStatus = iota
	LoginOK
	LoginClosed
)

func processConn(conn net.Conn, args string, player **Player) (LoginStatus, error) {
	username := strings.TrimSpace(args)

	if strings.ContainsFunc(username, func (r rune) bool {
		return !unicode.IsPrint(r)
	}) {
		//TODO: Need to comment about custom error in readme
		return LoginFailed, InvCharInNameErr
	} else if len(username) > 10 {
		//TODO: Need to comment about custom error in readme
		return LoginFailed, NameTooLongErr
	} else if len(username) < 2 {
		return LoginFailed, NameTooShortErr
	}

	onlinePlayersMu.Lock()
	defer onlinePlayersMu.Unlock()

	_, isOnline := onlinePlayers[username]
	if isOnline {
		return LoginFailed, NameInUseErr
	}
	*player = NewPlayer(username, "J'SAIS PAS", conn)
	onlinePlayers[username] = *player
	return LoginOK, nil
}

func handleLogin(conn net.Conn, command, args string, loginState *LoginStatus, player **Player) {
	command = strings.ToUpper((command))

	switch command {
		case "CONNECT":
			if args == "" {
				fmt.Fprintln(conn, MissingArgsErr.Error())
				*loginState = LoginFailed
			}

			result, err := processConn(conn, args, player)

			if err != nil {
				fmt.Fprintln(conn, err.Error())
			} else {
				fmt.Fprintln(conn, "OK connected")
			}
			
			*loginState = result
		case "QUIT":
			fmt.Fprintln(conn, "OK bye")
			*loginState = LoginClosed
			conn.Close()
		default:
			fmt.Fprintln(conn, InvalidCommandErr.Error())
	}
}

func commandDispatch(conn net.Conn, command, args string, loginState *LoginStatus, player *Player) {
	switch command {
		case "CONNECT": fmt.Fprintln(conn, AlreadyConnErr.Error())
		case "LOOK", "QUIT", "WHO", "STATUS", "INVENTORY", "QUESTS":
			if args != "" {
				//TODO: Need to comment about custom error in readme
				fmt.Fprintln(conn, UnknownArgsErr.Error())
			} else {
				if command == "QUIT" {
					fmt.Fprintln(conn, "OK bye")
					*loginState = LoginClosed
					conn.Close()
				} else if command == "LOOK" {
					function here
				} else if command == "WHO" {
					onlinePlayersMu.Lock()
					defer onlinePlayersMu.Unlock()
					fmt.Fprintf(conn, "OK players=%d\n", len(onlinePlayers))
				} else if command == "STATUS" {
					player.PlayerMu.Lock()
					defer player.PlayerMu.Unlock()
					playerStats := PlayerStatusResponse{
						HP: player.CurrHP,
						MaxHP: player.MaxHP,
						Status: player.Status,
					}
					statPrint, err := json.Marshal(playerStats)

					if err != nil {
						fmt.Fprintln(conn, JSONErr.Error())
						return
					}

					fmt.Fprintln(conn, "OK", string(statPrint))
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
						return
					}

					fmt.Fprintln(conn, "OK", string(sac))
				} else if command == "QUESTS" {
					function here
				}
			}
		case "MOVE", "CHAT", "GROUP", "TAKE", "DROP", "TALK", "ATTACK", "QUEST":
			if args == "" {
				//TODO: Need to comment about custom error in readme
				fmt.Fprintln(conn, MissingArgsErr.Error())
			} else {
				if command == "MOVE" {
					function here
				} else if command == "CHAT" {
					function here
				} else if command == "GROUP" {
					subparts := strings.SplitN(args, " ", 2)
					var subargs string
					if len(subparts) > 1 {
						subargs = parts[1]
					}
					groupFuncDispatcher(subparts[0], subargs, player)
				} else if command == "TAKE" {
					function here
				} else if command == "DROP" {
					function here
				} else if command == "TALK" {
					function here
				} else if command == "ATTACK" {
					function here
				} else if command == "QUEST" {
					function here
				}
			}
		default:
			//TODO: Need to comment about custom error in readme
			fmt.Fprintln(conn, InvalidCommandErr.Error())
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
		log.Fatal(ConnFailedErr.Error())
	}
	defer listener.Close()

	var currRetries int
	maxRetries := 5

	for {
		conn, err := listener.Accept()
		if err != nil {
			currRetries++
			if currRetries >= maxRetries {
				log.Fatal(ConnFailedErr.Error())
			} else {
				log.Println(InboundConnErr.Error())
				continue
			}
		} else {
			currRetries = 0
		}
		go handleTCPConn(conn)

	}
}
