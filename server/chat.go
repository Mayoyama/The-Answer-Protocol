package main

import (
	"fmt"
	"log/slog"
)

func chatDispatcher(scope, message string, player *Player) {
	pname := player.getPlayerName()

	switch scope {
	case "GLOBAL":
		onlinePlayersMu.Lock()
		defer onlinePlayersMu.Unlock()

		for _, p := range onlinePlayers {
			if p.getPlayerName() != pname {
				fmt.Fprintln(p.Conn, EvtGlobalChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "message", EvtGlobalChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	case "ROOM":
		playerLoc := player.getZoneID()
		area, ok := getZoneObj(playerLoc)

		if !ok {
			fmt.Fprintln(player.Conn, InternalErr.Error())
			slog.Error(InternalErr.Error(), "player", pname, "loc", playerLoc, "command", "CHAT", "scope", scope)
			player.Conn.Close()
			return
		}
		area.ZoneMu.Lock()
		defer area.ZoneMu.Unlock()

		for _, p := range area.InZone {
			if p.getPlayerName() != pname {
				fmt.Fprintln(p.Conn, EvtZoneChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "message", EvtZoneChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	case "GROUP":
		inGroup := player.getPlayerGroupInfo()

		if inGroup == nil {
			fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", pname, "command", "CHAT", "scope", scope)
			return
		}

		inGroup.GroupMu.Lock()
		defer inGroup.GroupMu.Unlock()

		for k, p := range inGroup.Members {
			if k != pname {
				fmt.Fprintln(p.Conn, EvtPartyChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "message", EvtPartyChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	default:
		fmt.Fprintln(player.Conn, InvalidArgsErr.Error())
		slog.Info(InvalidArgsErr.Error(), "player", pname, "command", "CHAT", "scope", scope)
	}

}
