package main

import (
	"fmt"
	"log/slog"
)

// chatDispatcher routes a CHAT command to the requested scope (GLOBAL, ROOM, or GROUP).
func chatDispatcher(scope, message string, player *Player, loginState *LoginStatus) {
	pname := player.getPlayerName()

	switch scope {
	case "GLOBAL":
		onlinePlayersMu.Lock()
		defer onlinePlayersMu.Unlock()

		for _, p := range onlinePlayers {
			if p.getPlayerName() != pname {
				_, _ = fmt.Fprintln(p.Conn, EvtGlobalChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "message", EvtGlobalChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	case "ROOM":
		playerLoc := player.getZoneID()
		area, ok := getZoneObj(playerLoc)

		if !ok {
			handleInternalError(player.Conn, InternalErr, loginState,
				slog.String("player", pname),
				slog.String("command", "CHAT"),
				slog.String("loc", playerLoc),
				slog.String("scope", scope),
			)

			return
		}

		area.ZoneMu.Lock()
		defer area.ZoneMu.Unlock()

		for _, p := range area.InZone {
			if p.getPlayerName() != pname {
				_, _ = fmt.Fprintln(p.Conn, EvtZoneChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "message", EvtZoneChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	case "GROUP":
		inGroup := player.getPlayerGroupInfo()

		if inGroup == nil {
			_, _ = fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", pname, "command", "CHAT", "scope", scope)

			return
		}

		inGroup.GroupMu.Lock()
		defer inGroup.GroupMu.Unlock()

		for k, p := range inGroup.Members {
			if k != pname {
				_, _ = fmt.Fprintln(p.Conn, EvtPartyChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "message", EvtPartyChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	default:
		_, _ = fmt.Fprintln(player.Conn, InvalidArgsErr.Error())
		slog.Info(InvalidArgsErr.Error(), "player", pname, "command", "CHAT", "scope", scope)
	}
}
