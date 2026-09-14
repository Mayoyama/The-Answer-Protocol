package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// onlinePlayers holds all currently connected players, keyed by username.
var (
	onlinePlayers   = make(map[string]*Player)
	onlinePlayersMu sync.Mutex
)

// Player represents a connected player and their in-game state.
type Player struct {
	Username       string
	MaxHP          int
	CurrHP         int
	CurrLoc        string
	Status         string
	GroupInfo      *Group
	Inventory      map[string]bool
	Conn           net.Conn
	TokenCount     float64
	BucketTS       time.Time
	TimeoutEnd     time.Time
	TimeoutActions int
	PlayerMu       sync.Mutex
}

// newPlayer creates a new Player at the given starting location.
func newPlayer(username, startLoc string, conn net.Conn) *Player {
	return &Player{
		Username:  username,
		MaxHP:     100,
		CurrHP:    100,
		CurrLoc:   startLoc,
		Inventory: make(map[string]bool),
		Conn:      conn,
	}
}

// cleanupPlayerData removes the player from their group, zone, and the online list on disconnect.
func (p *Player) cleanupPlayerData() {
	inPT := p.getPlayerGroupInfo()
	pname := p.getPlayerName()

	if inPT != nil {
		_ = inPT.groupLeave(p)
	}

	currLoc := p.getZoneID()
	zone, zOK := getZoneObj(currLoc)

	if zOK {
		zone.ZoneMu.Lock()
		delete(zone.InZone, pname)

		for _, pl := range zone.InZone {
			_, _ = fmt.Fprintln(pl.Conn, EvtZoneLeave(pname))
			slog.Info("SYS_MESSAGE", "player", pl.getPlayerName(), "message", EvtZoneLeave(pname), "reason", "player cleanup")
		}

		for i, b := range p.Inventory {
			if b {
				item, ok := resolveItem(i)

				if !ok {
					continue
				}

				if item.BaseLoc == zone {
					zone.Items[i] = item

				} else {
					item.BaseLoc.ZoneMu.Lock()
					item.BaseLoc.Items[i] = item
					item.BaseLoc.ZoneMu.Unlock()
				}
			}
		}

		zone.ZoneMu.Unlock()
	}

	onlinePlayersMu.Lock()
	_, pOK := onlinePlayers[pname]

	if pOK {
		delete(onlinePlayers, pname)
	}

	onlinePlayersMu.Unlock()

	slog.Info("PLAYER_CLEANUP_COMPLETE", "player", pname)
}

// printStatus sends the player's HP/status as a JSON response.
func printStatus(player *Player, command, args string) {
	player.PlayerMu.Lock()

	playerStats := PlayerStatusResponse{
		HP:     player.CurrHP,
		MaxHP:  player.MaxHP,
		Status: player.Status,
	}

	statPrint, err := json.Marshal(playerStats)
	player.PlayerMu.Unlock()

	if err != nil {
		_, _ = fmt.Fprintln(player.Conn, JSONErr.Error())
		slog.Error(JSONErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)

		return
	}

	_, _ = fmt.Fprintln(player.Conn, "OK", string(statPrint))
	slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", "OK "+string(statPrint), "command", command)
}
