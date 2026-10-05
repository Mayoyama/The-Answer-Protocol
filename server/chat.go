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

		for wMem, p := range onlinePlayers {
			if wMem != pname {
				_, _ = fmt.Fprintln(p.Conn, EvtGlobalChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "recipient", wMem, "message", EvtGlobalChat(pname, message), "command", "CHAT", "scope", scope)
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

		for zMem, p := range area.InZone {
			if zMem != pname {
				_, _ = fmt.Fprintln(p.Conn, EvtZoneChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "recipient", zMem, "message", EvtZoneChat(pname, message), "command", "CHAT", "scope", scope)
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

		for gMem, p := range inGroup.Members {
			if gMem != pname {
				_, _ = fmt.Fprintln(p.Conn, EvtPartyChat(pname, message))
				slog.Info("SYS_MESSAGE", "player", pname, "recipient", gMem, "message", EvtPartyChat(pname, message), "command", "CHAT", "scope", scope)
			}
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "CHAT", "scope", scope)

	default:
		_, _ = fmt.Fprintln(player.Conn, InvalidArgsErr.Error())
		slog.Info(InvalidArgsErr.Error(), "player", pname, "command", "CHAT", "scope", scope)
	}
}

// announceBattleStartEnd sends a fight start/end message as the NPC to its room, skipping pname ("" = everyone).
func (n *NPC) announceBattleStartEnd(pname, message string) {
	nName := n.getNPCName()
	npcName := fmt.Sprintf("<<npc.%s>>", nName)

	n.BaseLoc.ZoneMu.Lock()
	defer n.BaseLoc.ZoneMu.Unlock()

	for zMem, p := range n.BaseLoc.InZone {
		if zMem != pname {
			_, _ = fmt.Fprintln(p.Conn, EvtZoneChat(npcName, message))
			slog.Info("SYS_MESSAGE", "NPC", nName, "recipient", zMem, "message", EvtZoneChat(npcName, message), "command", "CHAT", "scope", "ROOM")
		}
	}
}
