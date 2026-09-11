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

func commandDispatch(command, args string, loginState *LoginStatus, player *Player) {
	pname := player.getPlayerName()

	switch command {
	case "CONNECT":
		fmt.Fprintln(player.Conn, AlreadyConnErr.Error())
		slog.Info(AlreadyConnErr.Error(), "player", pname, "command", command, "args", args)

	case "LOOK", "QUIT", "WHO", "STATUS", "INVENTORY", "QUESTS":
		if args != "" {
			fmt.Fprintln(player.Conn, InvalidArgsErr.Error())
			slog.Info(InvalidArgsErr.Error(), "player", pname, "command", command, "args", nil)

		} else {
			switch command {
			case "QUIT":
				fmt.Fprintln(player.Conn, "OK bye")
				slog.Info("PLAYER_QUIT", "player", pname, "command", command)
				*loginState = LoginClosed
				player.Conn.Close()

			case "LOOK":
				handleLook(player, loginState)

			case "WHO":
				handleWho(player, loginState, command)

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
			fmt.Fprintln(player.Conn, MissingArgsErr.Error())
			slog.Info(MissingArgsErr.Error(), "player", pname, "command", command, "args", args)

		} else {
			subparts := strings.SplitN(args, " ", 2)
			var subargs string
			if len(subparts) > 1 {
				subargs = subparts[1]
			}

			switch command {
			case "MOVE":
				err, currLoc := handleMove(args, player)

				switch {
				case err == nil:
					//pass

				case err == InternalErr:
					handleInternalError(player.Conn, InternalErr, loginState,
						slog.String("player", pname),
						slog.String("command", "MOVE"),
						slog.String("oldLoc", currLoc),
						slog.String("direction", args),
					)

				default:
					fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "oldLoc", currLoc, "direction", args)
				}

			case "CHAT":
				chatDispatcher(subparts[0], subargs, player)

			case "GROUP":
				groupFuncDispatcher(subparts[0], subargs, player)

			case "TAKE":
				err := player.itemTake(args)

				if err != nil {
					fmt.Fprintln(player.Conn, err.Error())
					slog.Info("SYS_MESSAGE", "player", pname, "message", err.Error(), "command", command)
				}

			case "DROP":
				err := player.itemDrop(args)

				if err != nil {
					fmt.Fprintln(player.Conn, err.Error())
					slog.Info("SYS_MESSAGE", "player", pname, "message", err.Error(), "command", command)
				}

			case "TALK":
				err, loc := handleTalk(args, player)

				switch {
				case err == nil:
					//pass

				case err == InternalErr:
					handleInternalError(player.Conn, InternalErr, loginState,
						slog.String("player", pname),
						slog.String("command", command),
						slog.String("loc", loc),
						slog.String("npc", args),
					)

				default:
					fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "loc", loc, "npc", args)
				}

			case "ATTACK":
				//function here

			case "QUEST":
				//function here
			}
		}

	default:
		fmt.Fprintln(player.Conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "player", pname, "command", command, "args", args)
	}

}
