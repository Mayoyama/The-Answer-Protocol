package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
)

var (
	onlinePlayers   = make(map[string]*Player)
	onlinePlayersMu sync.Mutex
)

type Player struct {
	Username  string
	MaxHP     int
	CurrHP    int
	CurrLoc   string
	Status    string
	GroupInfo *Group
	Inventory map[string]bool
	Conn      net.Conn
	PlayerMu  sync.Mutex
}

func NewPlayer(username, startLoc string, conn net.Conn) *Player {
	return &Player{
		Username:  username,
		MaxHP:     100,
		CurrHP:    100,
		CurrLoc:   startLoc,
		Inventory: make(map[string]bool),
		Conn:      conn,
	}
}

func (p *Player) CleanupPlayerData() {
	inPT := p.getPlayerGroupInfo()
	pname := p.getPlayerName()

	if inPT != nil {
		inPT.GroupLeave(p)
	}

	currLoc := p.getZoneID()
	zone, zOK := getZoneObj(currLoc)

	if zOK {
		zone.ZoneMu.Lock()
		delete(zone.InZone, pname)
		for _, pl := range zone.InZone {
			fmt.Fprintln(pl.Conn, EvtZoneLeave(pname))
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

	slog.Info("SYSTEM_INFO: Player cleanup complete", "player", pname)
}

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
		fmt.Fprintln(player.Conn, JSONErr.Error())
		slog.Error(JSONErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)
		return
	}

	fmt.Fprintln(player.Conn, "OK", string(statPrint))
	slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", "OK "+string(statPrint), "command", command)

}
