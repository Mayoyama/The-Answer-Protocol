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
			if p.getPlayerName() == pname {
				continue
			}
			fmt.Fprintln(p.Conn, EvtGlobalChat(pname, message))
		}

		fmt.Fprintln(player.Conn, "OK")

	case "ROOM":
		playerLoc := player.getZoneID()
		area, ok := getZoneObj(playerLoc)

		if !ok {
			fmt.Fprintln(player.Conn, InternalErr.Error())
			slog.Error(InternalErr.Error(), "player", pname, "loc", playerLoc)
			player.Conn.Close()
			return
		}
		area.ZoneMu.Lock()
		defer area.ZoneMu.Unlock()

		for _, p := range area.InZone {
			if p.getPlayerName() != pname {
				fmt.Fprintln(p.Conn, EvtZoneChat(pname, message))
			}
		}

		fmt.Fprintln(player.Conn, "OK")

	case "GROUP":
		inGroup := player.getPlayerGroupInfo()

		if inGroup == nil {
			fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", pname)
			return
		}

		inGroup.GroupMu.Lock()
		defer inGroup.GroupMu.Unlock()

		for k, p := range inGroup.Members {
			if k != pname {
				fmt.Fprintln(p.Conn, EvtPartyChat(pname, message))
			}
		}

		fmt.Fprintln(player.Conn, "OK")
	}
}
