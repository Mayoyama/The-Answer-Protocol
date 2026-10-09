package main

import (
	"encoding/json"
	"fmt"
)

// examineDispatcher handles EXAMINE <NPC|ITEM|KEYITEM> <name>, replying with the target's name and description.
func examineDispatcher(scope, target string, player *Player) (string, error) {
	var exResponse ExamineResponse
	currLoc := player.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	switch scope {
	case "NPC":
		npc, ok := currZone.getNPCObject(target)

		if !ok {
			return currLoc, NPCNotFoundErr
		}

		npcRole := npc.getNPCRole()

		npc.NPCMu.Lock()

		exResponse = ExamineResponse{
			Name:        npc.NPCName,
			Description: npc.Description,
			NPCRole:     npcRole,
		}

		npc.NPCMu.Unlock()

	case "ITEM":
		i, ok := resolveItem(target)

		if !ok {
			return currLoc, ItemNotFoundErr
		}

		currZone.ZoneMu.Lock()
		_, inRoom := currZone.Items[i.ItemID]
		currZone.ZoneMu.Unlock()

		player.PlayerMu.Lock()
		inBag := player.Inventory[i.ItemID]
		player.PlayerMu.Unlock()

		if !inRoom && !inBag {
			return currLoc, ItemNotFoundErr
		}

		exResponse = ExamineResponse{
			Name:        i.ItemName,
			Description: i.Description,
			Obtainable:  i.Obtainable,
		}

	case "KEYITEM":
		ki, ok := resolveKeyItem(target)

		if !ok {
			return currLoc, ItemNotFoundErr
		}

		player.PlayerMu.Lock()
		_, held := player.KeyInventory[ki.ItemID]
		player.PlayerMu.Unlock()

		if !held {
			return currLoc, NotInInvErr
		}

		exResponse = ExamineResponse{
			Name:        ki.ItemName,
			Description: ki.Description,
		}

	default:
		return currLoc, InvalidArgsErr
	}

	exResString, err := json.Marshal(exResponse)

	if err != nil {
		return currLoc, JSONErr
	}

	_, _ = fmt.Fprintln(player.Conn, "OK "+string(exResString))

	return "", nil
}
