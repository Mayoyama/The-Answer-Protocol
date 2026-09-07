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

		for p := range onlinePlayers {
			if onlinePlayers[p].Username == pname {
				continue
			}
			channel := onlinePlayers[p].Conn
			fmt.Fprintln(channel, EvtGlobalChat(pname, message))
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

		for p := range area.InZone {
			if area.InZone[p].Username == pname {
				continue
			}
			channel := area.InZone[p].Conn
			fmt.Fprintln(channel, EvtZoneChat(pname, message))
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

		for member := range inGroup.Members {
			if member == pname {
				continue
			}
			channel := inGroup.Members[member].Conn
			fmt.Fprintln(channel, EvtPartyChat(pname, message))
		}

		fmt.Fprintln(player.Conn, "OK")
	}
}
