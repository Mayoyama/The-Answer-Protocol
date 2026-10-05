package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// npcs holds all loaded NPCs, keyed by their world.yaml key.
var (
	npcs   = make(map[string]*NPC)
	npcsMu sync.Mutex
)

// NPCRole categorizes an NPC's behavior (general, quest giver, healer, enemy).
type NPCRole int

// NPC role values.
const (
	General NPCRole = iota
	QuestGiver
	Healer
	Enemy
)

// NPC represents a non-player character in the world.
type NPC struct {
	NPCID          string
	NPCName        string
	Description    string
	Dialogue       []string
	BattleDialogue BattleDialogue
	HealDialogue   string
	ChatIndex      int
	Quests         map[string]*Quest
	Role           NPCRole
	Attackable     bool
	Occupied       bool
	Stats          NPCStats
	BaseLoc        *Zone
	NPCMu          sync.Mutex
}

// NPCStats holds an NPC's HP and combat stats from world.yaml.
type NPCStats struct {
	HP          int `yaml:"hp"`
	Strength    int `yaml:"str"`
	BattleSkill int `yaml:"battle_skill"`
	Dexterity   int `yaml:"dex"`
}

// BattleDialogue holds an enemy's fight start, player-victory and NPC-victory lines.
type BattleDialogue struct {
	BattleStart   string `yaml:"battle_start"`
	PlayerVictory string `yaml:"player_victory"`
	NPCVictory    string `yaml:"npc_victory"`
}

// resolveNPC finds an NPC by ID, name or world.yaml key (case-insensitive).
func resolveNPC(input string) (*NPC, bool) {
	npcsMu.Lock()
	defer npcsMu.Unlock()

	for k, spawn := range npcs {
		if strings.EqualFold(spawn.NPCID, input) || strings.EqualFold(spawn.NPCName, input) || strings.EqualFold(k, input) {
			return spawn, true
		}
	}

	return nil, false
}

// getDialogue advances a matching talk_to quest step (checking, granting and taking key items)
// and returns its dialogue; otherwise it cycles the NPC's normal lines.
func (n *NPC) getDialogue(player *Player) (*PlayerQuest, string, bool) {
	player.PlayerMu.Lock()

	for _, pq := range player.Quests {
		if pq.Status != Active {
			continue
		}

		step, ok := pq.Quest.Steps[pq.StepIndex].(*TalkToAction)

		if !ok {
			continue
		}

		npcsMu.Lock()
		npc, found := npcs[step.Target]
		npcsMu.Unlock()

		if found && npc == n {
			hasAllItems := true

			for _, ki := range step.ReceivesKeyItems {
				keyItem, ok := resolveKeyItem(ki)

				if !ok || !player.KeyInventory[keyItem.ItemID] {
					hasAllItems = false

					break
				}
			}

			if !hasAllItems {
				if step.MissingItemsDialogue != "" {
					player.PlayerMu.Unlock()

					return nil, step.MissingItemsDialogue, false
				}

				continue
			}

			pq.StepIndex++

			var grantedItems []string

			for _, ki := range step.GrantsKeyItems {
				keyItem, ok := resolveKeyItem(ki)

				if !ok {
					msg := "You receive a quest key item, but it disintegrates before your eyes."
					_, _ = fmt.Fprintln(player.Conn, msg)
					slog.Error(ItemNotFoundErr.Error(), "key_item", ki, "quest", pq.Quest.Name, "player", player.Username, "npc", step.Target, "message", msg)

					continue
				}

				grantedItems = append(grantedItems, keyItem.ItemName)
				player.KeyInventory[keyItem.ItemID] = true
			}

			for _, ki := range step.ReceivesKeyItems {
				keyItem, ok := resolveKeyItem(ki)

				if !ok {
					msg := "You seem to have lost a quest key item, so you hand over a random object hoping the ignorant NPC won't notice."
					_, _ = fmt.Fprintln(player.Conn, msg)
					slog.Error(ItemNotFoundErr.Error(), "key_item", ki, "quest", pq.Quest.Name, "player", player.Username, "npc", step.Target, "message", msg)

					continue
				}

				delete(player.KeyInventory, keyItem.ItemID)
			}

			complete := pq.StepIndex >= len(pq.Quest.Steps)

			player.PlayerMu.Unlock()

			if len(grantedItems) > 0 {
				return pq, fmt.Sprintf("%s (You receive %s)", step.Dialogue, strings.Join(grantedItems, ", ")), complete
			}

			return pq, step.Dialogue, complete

		} else {
			continue
		}
	}

	player.PlayerMu.Unlock()

	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()

	chats := len(n.Dialogue)

	if chats == 0 {
		return nil, "...", false //default dialogue if no dialogue is set
	}

	option := n.ChatIndex % chats

	n.ChatIndex++

	return nil, n.Dialogue[option], false
}

// handleTalk heals the player if the NPC is a healer, otherwise sends its dialogue and
// completes finished quests.
func handleTalk(target string, player *Player) (string, error) {
	pname := player.getPlayerName()
	pHP := player.getPlayerHP()
	pMaxHP := player.getPlayerMaxHP()
	currLoc := player.getZoneID()
	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	spawn, ok := currZone.getNPCObject(target)

	if !ok {
		return currLoc, NPCNotFoundErr
	}

	sname := spawn.getNPCName()

	if npcRole := spawn.getNPCRole(); npcRole == "healer" && pHP < pMaxHP {
		healText := spawn.getNPCHealerString()
		_ = player.restoreSelfHP(pMaxHP)

		if err := player.setPlayerHPStatus(); err != nil {
			player.PlayerMu.Lock()
			player.Status = Unknown
			player.PlayerMu.Unlock()

			slog.Warn(err.Error(), "player", pname, "command", "TALK", "npc", sname)
		}

		healMessage := fmt.Sprintf("OK %s", healText)

		_, _ = fmt.Fprintln(player.Conn, healMessage)
		slog.Info("SYS_MESSAGE", "player", pname, "message", healMessage, "command", "TALK", "NPC", sname, "loc", currLoc)

		return "", nil
	}

	pq, dialogue, completedQuest := spawn.getDialogue(player)

	_, _ = fmt.Fprintln(player.Conn, "OK "+dialogue)
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+dialogue, "command", "TALK", "NPC", sname, "loc", currLoc)

	if completedQuest {
		pq.completeQuest(player)
	}

	return "", nil
}
