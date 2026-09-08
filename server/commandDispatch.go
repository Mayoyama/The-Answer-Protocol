package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

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
			switch command {
			case "QUIT":
				fmt.Fprintln(conn, "OK bye")
				slog.Info("PLAYER_QUIT", "player", player.Username, "command", command)
				*loginState = LoginClosed
				conn.Close()

			case "LOOK":
				//function here

			case "WHO":
				onlinePlayersMu.Lock()
				defer onlinePlayersMu.Unlock()
				fmt.Fprintf(conn, "OK players=%d\n", len(onlinePlayers))
				slog.Info("SYS_MESSAGE", "player", player.Username, "message", "OK players="+strconv.Itoa(len(onlinePlayers)), "command", command)

			case "STATUS":
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

			case "INVENTORY":
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

			case "QUESTS":
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

			switch command {
			case "MOVE":
				//function here

			case "CHAT":
				chatDispatcher(subparts[0], subargs, player)

			case "GROUP":
				groupFuncDispatcher(subparts[0], subargs, player)

			case "TAKE":
				//function here

			case "DROP":
				//function here

			case "TALK":
				//function here

			case "ATTACK":
				//function here

			case "QUEST":
				//function here
			}
		}

	default:
		fmt.Fprintln(conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "player", player.Username, "command", command, "args", args)
	}

}
