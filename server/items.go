package main

import (
	"fmt"
	"log/slog"
	"sync"
)

var (
	items   = make(map[string]*Item)
	itemsMu sync.Mutex
)

type Item struct {
	ItemID      string
	ItemName    string
	Description string
	Obtainable  bool
	BaseLoc     *Zone
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

func (p *Player) itemTake(itemName string) error {

	itemsMu.Lock()
	i, ok := items[itemName]
	itemsMu.Unlock()
	if !ok {
		return ItemNotFoundErr
	}

	currLoc := p.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return InternalErr
	}

	currZone.ZoneMu.Lock()
	if _, ok := currZone.Items[itemName]; !ok {
		currZone.ZoneMu.Unlock()
		return ItemNotFoundErr
	} else {
		delete(currZone.Items, itemName)
	}
	currZone.ZoneMu.Unlock()

	p.AddItem(itemName)

	fmt.Fprintln(p.Conn, "OK taken="+i.ItemID)
	slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", "OK taken="+i.ItemID, "command", "TAKE")

	return nil
}

func (p *Player) itemDrop(itemName string) error {

	itemsMu.Lock()
	i, ok := items[itemName]
	itemsMu.Unlock()
	if !ok {
		return ItemNotFoundErr
	}

	currLoc := p.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return InternalErr
	}

	p.PlayerMu.Lock()
	inBag := p.Inventory[itemName]
	p.PlayerMu.Unlock()

	if !inBag {
		return NotInInvErr
	} else {
		p.DropItem(itemName)
	}

	currZone.ZoneMu.Lock()
	currZone.Items[itemName] = i
	currZone.ZoneMu.Unlock()

	fmt.Fprintln(p.Conn, "OK dropped="+i.ItemID)
	slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", "OK dropped="+i.ItemID, "command", "DROP")

	return nil
}
