package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// zones holds all loaded zones, keyed by their world.yaml key.
var (
	zones   = make(map[string]*Zone)
	zonesMu sync.Mutex
)

// ZoneMoveDirection is one of the six movement directions.
type ZoneMoveDirection int

// Direction values; InvalidDirection is the zero value.
const (
	InvalidDirection ZoneMoveDirection = iota
	Up
	Down
	North
	East
	South
	West
)

// Zone represents a location in the world.
type Zone struct {
	ZoneID      string
	ZoneName    string
	Description string
	Exits       map[ZoneMoveDirection]string
	InZone      map[string]*Player
	Items       map[string]*Item
	NPCs        map[string]*NPC
	ZoneMu      sync.Mutex
}

// parseDirection converts a direction word to its enum (case-insensitive).
func parseDirection(direction string) (ZoneMoveDirection, bool) {
	direction = strings.ToLower(direction)

	switch direction {
	case "up":
		return Up, true
	case "down":
		return Down, true
	case "north":
		return North, true
	case "east":
		return East, true
	case "south":
		return South, true
	case "west":
		return West, true
	default:
		return InvalidDirection, false
	}
}

// String returns the direction's lowercase name.
func (d ZoneMoveDirection) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	case North:
		return "north"
	case East:
		return "east"
	case South:
		return "south"
	case West:
		return "west"
	default:
		return "unknown"
	}
}

// handleWho sends the player a JSON count of players in their room and on the server.
func handleWho(player *Player, loginState *LoginStatus, command string) {
	pname := player.getPlayerName()

	onlinePlayersMu.Lock()
	onlineCount := len(onlinePlayers)
	onlinePlayersMu.Unlock()

	currLoc := player.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		handleInternalError(player.Conn, InternalErr, loginState,
			slog.String("player", pname),
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
		_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "err", err, "player", pname, "command", "WHO")

		return
	}

	_, _ = fmt.Fprintf(player.Conn, "OK %s\n", string(info))
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+string(info), "command", command)
}

// handleLook sends the player a JSON description of their current room.
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

	var exits = make(map[string]string)

	for dir, target := range area.Exits {
		exits[dir.String()] = target
	}

	roomInfo := RoomInfo{
		RoomID:      area.ZoneID,
		Name:        area.ZoneName,
		Description: area.Description,
		Exits:       exits,
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
		_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "err", err, "player", pname, "command", "LOOK")
		return
	}

	_, _ = fmt.Fprintf(player.Conn, "OK %s\n", string(info))
	slog.Info("SYS_MESSAGE", "player", pname, "message", string(info), "command", "LOOK")
}

// shiftZone moves the player between zones, sending leave/enter events, and returns the new zone ID.
func shiftZone(player *Player, pname, newLoc string, fromZone, newZone *Zone) string {
	fromZone.ZoneMu.Lock()
	delete(fromZone.InZone, pname)

	for _, p := range fromZone.InZone {
		_, _ = fmt.Fprintln(p.Conn, EvtZoneLeave(pname))
		slog.Info(EvtZoneLeave(pname), "loc", fromZone.ZoneID)
	}

	fromZone.ZoneMu.Unlock()

	player.PlayerMu.Lock()
	player.CurrLoc = newLoc
	player.PlayerMu.Unlock()

	newZone.ZoneMu.Lock()
	newZoneID := newZone.ZoneID
	newZone.InZone[pname] = player

	for k, p := range newZone.InZone {
		if p != player {
			_, _ = fmt.Fprintln(p.Conn, EvtZoneEnter(pname))
			slog.Info(EvtZoneEnter(pname), "recipient", k, "loc", newZone.ZoneName)
		}
	}

	newZone.ZoneMu.Unlock()

	return newZoneID
}

// handleMove moves the player through an exit to an adjacent zone.
func handleMove(direction string, player *Player) (string, error) {
	pname := player.getPlayerName()

	currLoc := player.getZoneID()
	zone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	direction = strings.ToLower(direction)
	moveDir, ok := parseDirection(direction)

	if !ok {
		return currLoc, NoExitErr
	}

	zone.ZoneMu.Lock()
	newLoc, ok := zone.Exits[moveDir]
	zone.ZoneMu.Unlock()

	if !ok {
		return currLoc, NoExitErr
	}

	newZone, ok := getZoneObj(newLoc)

	if !ok {
		return currLoc, InternalErr
	}

	newZoneID := shiftZone(player, pname, newLoc, zone, newZone)

	_, _ = fmt.Fprintln(player.Conn, "OK room="+newZoneID)
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK room="+newZoneID, "command", "MOVE", "prev_loc", currLoc)

	player.PlayerMu.Lock()

	for _, pq := range player.Quests {
		if pq.Status != Active {
			continue
		}

		step, ok := pq.Quest.Steps[pq.StepIndex].(*EnterAreaAction)

		if !ok {
			continue
		}

		if newLoc == step.Area {
			_, _ = fmt.Fprintln(player.Conn, step.Message)
			slog.Info("SYS_MESSAGE", "player", pname, "message", step.Message, "action", "EnterAreaAction", "loc", newLoc)

			pq.StepIndex++
			questComplete := pq.StepIndex >= len(pq.Quest.Steps)

			player.PlayerMu.Unlock()

			if questComplete {
				pq.completeQuest(player)
			}

			return "", nil
		}
	}

	player.PlayerMu.Unlock()

	return "", nil
}

// respawnPlayer moves a defeated player to respawnZone and sends EVT RESPAWN.
func respawnPlayer(player *Player) (string, error) {
	pname := player.getPlayerName()
	currLoc := player.getZoneID()

	zone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	newZone, ok := getZoneObj(respawnZone)

	if !ok {
		return currLoc, InternalErr
	}

	newZoneID := shiftZone(player, pname, respawnZone, zone, newZone)

	_, _ = fmt.Fprintln(player.Conn, EvtPlayerRespawn())
	slog.Info("SYS_MESSAGE", "player", pname, "message", EvtPlayerRespawn(), "reason", "playerDeath", "prev_loc", currLoc, "curr_loc", newZoneID)

	return "", nil
}
