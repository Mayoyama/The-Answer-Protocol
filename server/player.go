package main

import (
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
