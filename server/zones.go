package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

var (
	zones   = make(map[string]*Zone)
	zonesMu sync.Mutex
)

type Zone struct {
	ZoneID      string
	ZoneName    string
	Description string
	Exits       map[string]string
	InZone      map[string]*Player
	Items       map[string]*Item
	NPCs        map[string]*NPC
	ZoneMu      sync.Mutex
}

func handleWho(player *Player, loginState *LoginStatus, command string) {
	onlinePlayersMu.Lock()
	onlineCount := len(onlinePlayers)
	onlinePlayersMu.Unlock()

	currLoc := player.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		handleInternalError(player.Conn, InternalErr, loginState,
			slog.String("player", player.getPlayerName()),
			slog.Any("loc", currLoc),
			slog.String("command", "WHO"),
		)
		return
	}

	var roomPl []string
	currZone.ZoneMu.Lock()
	for _, p := range currZone.InZone {
		roomPl = append(roomPl, p.getPlayerName())
	}
	currZone.ZoneMu.Unlock()

	whoResp := WhoResponse{
		RoomPlayers: roomPl,
		ServerCount: onlineCount,
	}

	info, err := json.Marshal(whoResp)

	if err != nil {
		fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "err", err, "player", player.getPlayerName(), "command", "WHO")
		return
	}

	fmt.Fprintf(player.Conn, "OK %s\n", string(info))
	slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", "OK"+string(info), "command", command)
}

func handleLook(player *Player, loginState *LoginStatus) {
	pname := player.getPlayerName()
	currLoc := player.getZoneID()

	if currLoc == "" {
		handleInternalError(player.Conn, InternalErr, loginState,
			slog.String("player", pname),
			slog.Any("loc", nil),
			slog.String("command", "LOOK"),
		)
		return
	}

	area, ok := getZoneObj(currLoc)

	if !ok {
		handleInternalError(player.Conn, InternalErr, loginState,
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
		players = append(players, p.getPlayerName())
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
		fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "err", err, "player", pname, "command", "LOOK")
		return
	}

	fmt.Fprintf(player.Conn, "OK %s\n", string(info))
	slog.Info("SYS_MESSAGE", "player", pname, "message", string(info), "command", "LOOK")
}

func handleMove(direction string, player *Player) (error, string) {
	pname := player.getPlayerName()
	currLoc := player.getZoneID()
	zone, ok := getZoneObj(currLoc)
	if !ok {
		return InternalErr, currLoc
	}

	zone.ZoneMu.Lock()
	newLoc, ok := zone.Exits[direction]
	zone.ZoneMu.Unlock()
	if !ok {
		return NoExitErr, currLoc
	}

	newZone, ok := getZoneObj(newLoc)
	if !ok {
		return InternalErr, currLoc
	}

	zone.ZoneMu.Lock()
	delete(zone.InZone, pname)
	for _, p := range zone.InZone {
		fmt.Fprintln(p.Conn, EvtZoneLeave(pname))
		slog.Info(EvtZoneLeave(pname), "loc", currLoc)
	}
	zone.ZoneMu.Unlock()

	player.PlayerMu.Lock()
	player.CurrLoc = newLoc
	player.PlayerMu.Unlock()

	newZone.ZoneMu.Lock()
	newZoneID := newZone.ZoneID
	newZone.InZone[pname] = player
	for _, p := range newZone.InZone {
		if p != player {
			fmt.Fprintln(p.Conn, EvtZoneEnter(pname))
			slog.Info(EvtZoneEnter(pname), "loc", newZone.ZoneName)
		}
	}
	newZone.ZoneMu.Unlock()

	fmt.Fprintln(player.Conn, "OK room="+newZoneID)
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK room="+newZoneID, "command", "MOVE", "prev_loc", currLoc)

	return nil, ""
}
