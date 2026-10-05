package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// items holds all loaded items, keyed by their world.yaml key.
// keyItems holds all loaded key items, keyed by their world.yaml key.
var (
	items    = make(map[string]*Item)
	itemsMu  sync.Mutex
	keyItems = make(map[string]*KeyItem)
)

// Item represents a pickable object in the world.
type Item struct {
	ItemID      string
	ItemName    string
	Description string
	Obtainable  bool
	BaseLoc     *Zone
}

// KeyItem is a per-player quest item, kept separate from world items.
type KeyItem struct {
	ItemID      string
	ItemName    string
	Description string
	SkillBoost  int
}

// resolveItem finds an item by ID, name or world.yaml key (case-insensitive).
func resolveItem(input string) (*Item, bool) {
	itemsMu.Lock()
	defer itemsMu.Unlock()

	for k, item := range items {
		if strings.EqualFold(item.ItemID, input) || strings.EqualFold(item.ItemName, input) || strings.EqualFold(k, input) {
			return item, true
		}
	}

	return nil, false
}

// resolveKeyItem finds a key item by ID, name or world.yaml key (case-insensitive).
func resolveKeyItem(input string) (*KeyItem, bool) {
	for k, ki := range keyItems {
		if strings.EqualFold(ki.ItemID, input) || strings.EqualFold(ki.ItemName, input) || strings.EqualFold(k, input) {
			return ki, true
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
	defer currZone.ZoneMu.Unlock()

	_, exists := currZone.Items[i.ItemID]

	if !exists {
		return currLoc, ItemNotFoundErr
	}

	if !i.Obtainable {
		return currLoc, NotObtainableErr
	}

	delete(currZone.Items, i.ItemID)

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

	pname := player.Username

	bag := make([]string, 0, len(player.Inventory))

	for item := range player.Inventory {
		bag = append(bag, item)
	}

	player.PlayerMu.Unlock()

	sac, err := json.Marshal(bag)

	if err != nil {
		_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "player", pname, "command", command, "args", args)

		return
	}

	_, _ = fmt.Fprintln(player.Conn, "OK", string(sac))
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+string(sac), "command", command)
}

// printKIs sends the player's key items as a JSON response.
func printKIs(player *Player, command, args string) {
	player.PlayerMu.Lock()

	pname := player.Username

	kItems := make([]string, 0, len(player.KeyInventory))

	for item := range player.KeyInventory {
		kItems = append(kItems, item)
	}

	player.PlayerMu.Unlock()

	sac, err := json.Marshal(kItems)

	if err != nil {
		_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "player", pname, "command", command, "args", args)

		return
	}

	_, _ = fmt.Fprintln(player.Conn, "OK", string(sac))
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+string(sac), "command", command)
}
