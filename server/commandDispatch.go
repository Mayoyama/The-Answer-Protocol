package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// handleInternalError reports an internal error to the client and closes the connection.
func handleInternalError(conn net.Conn, err error, loginState *LoginStatus, slogKWARGS ...slog.Attr) {
	_, _ = fmt.Fprintln(conn, InternalErr.Error())
	slog.Default().LogAttrs(context.Background(), slog.LevelError, err.Error(), slogKWARGS...)

	*loginState = LoginClosed

	_ = conn.Close()
}

// playerInBattle reports whether the player is in an ongoing battle; if so, it also sends them COMMAND_NOT_AVAILABLE_IN_COMBAT and logs it.
func playerInBattle(conn net.Conn, command, pname string) bool {
	OngoingBattlesMu.Lock()
	_, inCombat := OngoingBattles[pname]
	OngoingBattlesMu.Unlock()

	if inCombat {
		_, _ = fmt.Fprintln(conn, CommandInCombatErr.Error())
		slog.Info(CommandInCombatErr.Error(), "player", pname, "command", command)

		return true
	}

	return false
}

// commandDispatch routes a post-login command to its handler, after rate-limit checks.
func commandDispatch(command, args string, loginState *LoginStatus, player *Player) {
	pname := player.getPlayerName()

	softBanned, timeRemaining := player.isOnTimeout(time.Now())
	if softBanned {
		player.handleSoftban(timeRemaining.Round(time.Second))

		return
	}

	hasTokens := player.handleTimeoutBucket(time.Now())

	if !hasTokens {
		_, _ = fmt.Fprintln(player.Conn, InputSpamErr.Error())
		slog.Warn("SYS_MESSAGE", "remote", player.Conn.RemoteAddr().String(), "player", pname, "message", InputSpamErr.Error())

		return
	}

	switch command {
	case "CONNECT":
		_, _ = fmt.Fprintln(player.Conn, AlreadyConnErr.Error())
		slog.Info(AlreadyConnErr.Error(), "player", pname, "command", command, "args", args)

	case "LOOK", "QUIT", "WHO", "STATUS", "INVENTORY", "QUESTS", "GOLD":
		if args != "" {
			_, _ = fmt.Fprintln(player.Conn, InvalidArgsErr.Error())
			slog.Info(InvalidArgsErr.Error(), "player", pname, "command", command, "args", nil)

		} else {
			switch command {
			case "QUIT":
				_, _ = fmt.Fprintln(player.Conn, "OK bye")
				slog.Info("PLAYER_QUIT", "player", pname, "command", command)

				*loginState = LoginClosed

				_ = player.Conn.Close()

			case "LOOK":
				handleLook(player, loginState)

			case "WHO":
				handleWho(player, loginState, command)

			case "STATUS":
				printStatus(player, command, args)

			case "INVENTORY":
				printInventory(player, command, args)

			case "QUESTS":
				printPlayerQuests(player)

			case "GOLD":
				printGoldBalance(player)
			}
		}

	case "MOVE", "CHAT", "GROUP", "TAKE", "DROP", "TALK", "ATTACK", "QUEST", "ACCEPT":
		if args == "" {
			_, _ = fmt.Fprintln(player.Conn, MissingArgsErr.Error())
			slog.Info(MissingArgsErr.Error(), "player", pname, "command", command, "args", args)

		} else {
			subparts := strings.SplitN(args, " ", 2)

			var subargs string

			if len(subparts) > 1 {
				subargs = subparts[1]
			}

			switch command {
			case "MOVE":
				if playerInBattle(player.Conn, command, pname) {
					return
				}

				currLoc, err := handleMove(args, player)

				switch err {
				case nil:
				case InternalErr:
					handleInternalError(player.Conn, InternalErr, loginState,
						slog.String("player", pname),
						slog.String("command", "MOVE"),
						slog.String("oldLoc", currLoc),
						slog.String("direction", args),
					)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "oldLoc", currLoc, "direction", args)
				}

			case "CHAT":
				chatDispatcher(subparts[0], subargs, player, loginState)

			case "GROUP":
				groupFuncDispatcher(subparts[0], subargs, player)

			case "TAKE":
				if playerInBattle(player.Conn, command, pname) {
					return
				}

				loc, err := player.itemTake(args)

				switch err {
				case nil:
				case InternalErr:
					handleInternalError(player.Conn, InternalErr, loginState,
						slog.String("player", pname),
						slog.String("command", command),
						slog.String("loc", loc),
						slog.String("item", args),
					)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "loc", loc, "item", args)
				}

			case "DROP":
				if playerInBattle(player.Conn, command, pname) {
					return
				}

				loc, err := player.itemDrop(args)

				switch err {
				case nil:
				case InternalErr:
					handleInternalError(player.Conn, InternalErr, loginState,
						slog.String("player", pname),
						slog.String("command", command),
						slog.String("loc", loc),
						slog.String("item", args),
					)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "loc", loc, "item", args)
				}

			case "TALK":
				loc, err := handleTalk(args, player)

				switch err {
				case nil:
				case InternalErr:
					handleInternalError(player.Conn, InternalErr, loginState,
						slog.String("player", pname),
						slog.String("command", command),
						slog.String("loc", loc),
						slog.String("npc", args),
					)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "loc", loc, "npc", args)
				}

			case "ATTACK":
				err := resolveAttackRequest(args, player)

				switch err {
				case nil:
				case JSONErr:
					_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
					slog.Error(JSONErr.Error(), "player", pname, "command", command, "args", args)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "npc", args)
				}

			case "FLEE":
				OngoingBattlesMu.Lock()
				battleField, inCombat := OngoingBattles[pname]
				OngoingBattlesMu.Unlock()

				if !inCombat {
					_, _ = fmt.Fprintln(player.Conn, InvalidCommandErr.Error())
					slog.Info(InvalidCommandErr.Error(), "player", pname, "command", command)

					return
				}

				ename := battleField.NPC.getNPCName()
				terminateBattle(pname, battleField.NPC)
				close(battleField.PlayerAttack)

				_, _ = fmt.Fprintln(player.Conn, "OK battle ended")
				slog.Info("SYS_MESSAGE", "player", pname, "message", "OK battle ended", "command", command, "npc", ename)

			case "QUEST":
				if playerInBattle(player.Conn, command, pname) {
					return
				}

				err := checkNPCQuest(player, args)

				switch err {
				case nil:
				case JSONErr:
					_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
					slog.Error(JSONErr.Error(), "player", pname, "command", command, "args", args)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "npc", args)
				}

			case "ACCEPT":
				if playerInBattle(player.Conn, command, pname) {
					return
				}

				err := acceptQuest(player, args)

				switch err {
				case nil:
				case JSONErr:
					_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
					slog.Error(JSONErr.Error(), "player", pname, "command", command, "args", args)

				default:
					_, _ = fmt.Fprintln(player.Conn, err.Error())
					slog.Info(err.Error(), "player", pname, "command", command, "npc", args)
				}
			}
		}

	default:
		_, _ = fmt.Fprintln(player.Conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "player", pname, "command", command, "args", args)
	}
}
