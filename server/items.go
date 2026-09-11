package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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

func resolveItem(input string) (*Item, bool) {
	itemsMu.Lock()
	defer itemsMu.Unlock()

	for _, item := range items {
		if strings.EqualFold(item.ItemID, input) || strings.EqualFold(item.ItemName, input) {
			return item, true
		}
	}
	return nil, false
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
	i, ok := resolveItem(itemName)
	if !ok {
		return ItemNotFoundErr
	}

	currLoc := p.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return InternalErr
	}

	currZone.ZoneMu.Lock()
	if _, ok := currZone.Items[i.ItemID]; !ok {
		currZone.ZoneMu.Unlock()
		return ItemNotFoundErr
	} else {
		delete(currZone.Items, i.ItemID)
	}
	currZone.ZoneMu.Unlock()

	p.AddItem(i.ItemID)

	fmt.Fprintln(p.Conn, "OK taken="+i.ItemID)
	slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", "OK taken="+i.ItemID, "command", "TAKE")

	return nil
}

func (p *Player) itemDrop(itemName string) error {
	i, ok := resolveItem(itemName)
	if !ok {
		return ItemNotFoundErr
	}

	currLoc := p.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return InternalErr
	}

	p.PlayerMu.Lock()
	inBag := p.Inventory[i.ItemID]
	p.PlayerMu.Unlock()

	if !inBag {
		return NotInInvErr
	} else {
		p.DropItem(i.ItemID)
	}

	currZone.ZoneMu.Lock()
	currZone.Items[i.ItemID] = i
	currZone.ZoneMu.Unlock()

	fmt.Fprintln(p.Conn, "OK dropped="+i.ItemID)
	slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", "OK dropped="+i.ItemID, "command", "DROP")

	return nil
}

func printInventory(player *Player, command, args string) {
	player.PlayerMu.Lock()
	bag := make([]string, 0, len(player.Inventory))

	for item := range player.Inventory {
		bag = append(bag, item)
	}

	sac, err := json.Marshal(bag)
	player.PlayerMu.Unlock()

	if err != nil {
		fmt.Fprintln(player.Conn, JSONErr.Error())
		slog.Error(JSONErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)
		return
	}

	fmt.Fprintln(player.Conn, "OK", string(sac))
	slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", "OK "+string(sac), "command", command)
}
