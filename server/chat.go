package main

import (
	"fmt"
	"log/slog"
)

func chatDispatcher(scope, message string, player *Player) {
	switch scope {
	case "GLOBAL":
		onlinePlayersMu.Lock()
		defer onlinePlayersMu.Unlock()

		player.PlayerMu.Lock()
		pname := player.Username
		player.PlayerMu.Unlock()

		for _, p := range onlinePlayers {
			if p.Username == pname {
				continue
			}
			fmt.Fprintln(p.Conn, EvtGlobalChat(pname, message))
		}

		fmt.Fprintln(player.Conn, "OK")

	case "ROOM":
		player.PlayerMu.Lock()
		playerLoc := player.CurrLoc
		pname := player.Username
		player.PlayerMu.Unlock()

		zonesMu.Lock()
		area, ok := zones[playerLoc]
		zonesMu.Unlock()

		if !ok {
			fmt.Fprintln(player.Conn, InternalErr.Error())
			slog.Error(InternalErr.Error(), "player", pname, "loc", playerLoc)
			player.Conn.Close()
			return
		}
		area.ZoneMu.Lock()
		defer area.ZoneMu.Unlock()

		for _, p := range area.InZone {
			if p.Username == pname {
				continue
			}
			fmt.Fprintln(p.Conn, EvtZoneChat(pname, message))
		}

		fmt.Fprintln(player.Conn, "OK")

	case "GROUP":
		player.PlayerMu.Lock()
		inGroup := player.GroupInfo
		pname := player.Username
		player.PlayerMu.Unlock()

		if inGroup == nil {
			fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", pname)
			return
		}

		inGroup.GroupMu.Lock()
		defer inGroup.GroupMu.Unlock()

		for k, p := range inGroup.Members {
			if k == pname {
				continue
			}
			fmt.Fprintln(p.Conn, EvtPartyChat(pname, message))
		}

		fmt.Fprintln(player.Conn, "OK")
	}
}
