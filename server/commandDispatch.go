package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
)

func handleInternalError(conn net.Conn, err error, loginState *LoginStatus, slogKWARGS ...slog.Attr) {
	fmt.Fprintln(conn, InternalErr.Error())
	slog.Default().LogAttrs(context.Background(), slog.LevelError, err.Error(), slogKWARGS...)
	*loginState = LoginClosed
	conn.Close()
}

func commandDispatch(conn net.Conn, command, args string, loginState *LoginStatus, player *Player) {
	switch command {
	case "CONNECT":
		fmt.Fprintln(conn, AlreadyConnErr.Error())
		slog.Info(AlreadyConnErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)

	case "LOOK", "QUIT", "WHO", "STATUS", "INVENTORY", "QUESTS":
		if args != "" {
			fmt.Fprintln(conn, InvalidArgsErr.Error())
			slog.Info(InvalidArgsErr.Error(), "player", player.getPlayerName(), "command", command, "args", nil)

		} else {
			switch command {
			case "QUIT":
				fmt.Fprintln(conn, "OK bye")
				slog.Info("PLAYER_QUIT", "player", player.getPlayerName(), "command", command)
				*loginState = LoginClosed
				conn.Close()

			case "LOOK":
				handleLook(player, loginState)

			case "WHO":
				handleWHO(player, loginState, command)

			case "STATUS":
				printStatus(player, command, args)

			case "INVENTORY":
				printInventory(player, command, args)

			case "QUESTS":
				//function here
			}
		}
	case "MOVE", "CHAT", "GROUP", "TAKE", "DROP", "TALK", "ATTACK", "QUEST":
		if args == "" {
			//TODO: Need to comment about custom error in readme
			fmt.Fprintln(conn, MissingArgsErr.Error())
			slog.Info(MissingArgsErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)

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
				err := player.itemTake(args)

				if err != nil {
					fmt.Fprintln(player.Conn, err.Error())
					slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", err.Error(), "command", command)
				}

			case "DROP":
				err := player.itemDrop(args)

				if err != nil {
					fmt.Fprintln(player.Conn, err.Error())
					slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", err.Error(), "command", command)
				}

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
		slog.Info(InvalidCommandErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)
	}

}
