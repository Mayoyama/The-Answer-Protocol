package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

func handleLook(player *Player, loginState *LoginStatus) {
	player.PlayerMu.Lock()
	pname := player.Username
	currLoc := player.CurrLoc
	conn := player.Conn
	player.PlayerMu.Unlock()

	if currLoc == "" {
		handleInternalError(conn, InternalErr, loginState,
			slog.String("player", pname),
			slog.Any("loc", nil),
			slog.String("command", "LOOK"),
		)
		return
	}

	zonesMu.Lock()
	area, ok := zones[currLoc]
	zonesMu.Unlock()

	if !ok {
		handleInternalError(conn, InternalErr, loginState,
			slog.String("player", pname),
			slog.String("loc", currLoc),
			slog.String("command", "LOOK"),
		)
		return
	}

	area.ZoneMu.Lock()
	roomInfo := RoomInfo{
		RoomID:      area.ZoneID,
		Name:        area.ZoneName,
		Description: area.Description,
		Exits:       area.Exits,
	}

	var players []string
	for _, p := range area.InZone {
		players = append(players, p.Username)
	}

	var items []string
	for _, item := range area.Items {
		items = append(items, item.ItemID)
	}

	var spawns []string
	for _, npc := range area.NPCs {
		spawns = append(spawns, npc.NPCID)
	}

	lookRes := LookResponse{
		Room:    roomInfo,
		Players: players,
		Items:   items,
		NPCS:    spawns,
	}
	area.ZoneMu.Unlock()

	info, err := json.Marshal(lookRes)

	if err != nil {
		fmt.Fprintln(conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "err", err, "player", pname, "command", "LOOK")
		return
	}

	fmt.Fprintf(conn, "OK %s\n", string(info))
	slog.Info("SYS_MESSAGE", "player", pname, "message", string(info), "command", "LOOK")
}
