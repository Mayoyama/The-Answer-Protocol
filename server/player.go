package main

import (
	"sync"
	"net"
)

var (
	onlinePlayers = make(map[string]*Player)
	onlinePlayersMu sync.Mutex
)

type Player struct {
	Username string
	MaxHP int
	CurrHP int
	CurrLoc string
	Status string
	GroupInfo *Group
	Inventory map[string]bool
	Conn net.Conn
	PlayerMu sync.Mutex
}

func NewPlayer(username, startLoc string, conn net.Conn) *Player {
	return &Player{
		Username: username,
		MaxHP: 100,
		CurrHP: 100,
		CurrLoc: startLoc,
		Inventory: make(map[string]bool),
		Conn: conn,
	}
}

func (p *Player) AddItem(itemID string) {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()
	p.Inventory[itemID] = true
}

func (p *Player) DropItem(itemID string) {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()
	delete(p.Inventory, itemID)
}