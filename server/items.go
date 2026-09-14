package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// items holds all loaded items, keyed by item ID.
var (
	items   = make(map[string]*Item)
	itemsMu sync.Mutex
)

// Item represents a pickable object in the world.
type Item struct {
	ItemID      string
	ItemName    string
	Description string
	Obtainable  bool
	BaseLoc     *Zone
}

// resolveItem finds an item by ID or name (case-insensitive).
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

// addItem adds an item to the player's inventory.
func (p *Player) addItem(itemID string) {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	p.Inventory[itemID] = true
}

// dropItem removes an item from the player's inventory.
func (p *Player) dropItem(itemID string) {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	delete(p.Inventory, itemID)
}

// itemTake moves an item from the player's current zone into their inventory.
func (p *Player) itemTake(itemName string) (string, error) {
	i, ok := resolveItem(itemName)

	if !ok {
		return "", ItemNotFoundErr
	}

	currLoc := p.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	currZone.ZoneMu.Lock()

	if _, ok := currZone.Items[i.ItemID]; !ok {
		currZone.ZoneMu.Unlock()
		return currLoc, ItemNotFoundErr

	} else {
		delete(currZone.Items, i.ItemID)
	}

	currZone.ZoneMu.Unlock()

	p.addItem(i.ItemID)

	_, _ = fmt.Fprintln(p.Conn, "OK taken="+i.ItemID)
	slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", "OK taken="+i.ItemID, "command", "TAKE")

	return "", nil
}

// itemDrop moves an item from the player's inventory into their current zone.
func (p *Player) itemDrop(itemName string) (string, error) {
	i, ok := resolveItem(itemName)

	if !ok {
		return "", ItemNotFoundErr
	}

	currLoc := p.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	p.PlayerMu.Lock()
	inBag := p.Inventory[i.ItemID]
	p.PlayerMu.Unlock()

	if !inBag {
		return "", NotInInvErr

	} else {
		p.dropItem(i.ItemID)
	}

	currZone.ZoneMu.Lock()
	currZone.Items[i.ItemID] = i
	currZone.ZoneMu.Unlock()

	_, _ = fmt.Fprintln(p.Conn, "OK dropped="+i.ItemID)
	slog.Info("SYS_MESSAGE", "player", p.getPlayerName(), "message", "OK dropped="+i.ItemID, "command", "DROP")

	return "", nil
}

// printInventory sends the player's inventory as a JSON response.
func printInventory(player *Player, command, args string) {
	player.PlayerMu.Lock()

	bag := make([]string, 0, len(player.Inventory))

	for item := range player.Inventory {
		bag = append(bag, item)
	}

	sac, err := json.Marshal(bag)

	player.PlayerMu.Unlock()

	if err != nil {
		_, _ = fmt.Fprintln(player.Conn, JSONErr.Error())
		slog.Error(JSONErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)

		return
	}

	_, _ = fmt.Fprintln(player.Conn, "OK", string(sac))
	slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", "OK "+string(sac), "command", command)
}
