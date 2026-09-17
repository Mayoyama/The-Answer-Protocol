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

// NPCRole categorizes an NPC's behavior (general, quest giver, enemy).
type NPCRole int

// NPC role values.
const (
	General NPCRole = iota
	QuestGiver
	Enemy
)

// NPC represents a non-player character in the world.
type NPC struct {
	NPCID       string
	NPCName     string
	Description string
	Dialogue    []string
	ChatIndex int
	Quests      map[string]*Quest
	Role        NPCRole
	Attackable  bool
	Stats       map[string]int
	BaseLoc     *Zone
	NPCMu       sync.Mutex
}

// resolveNPC finds an NPC by ID or name (case-insensitive).
func resolveNPC(input string) (*NPC, bool) {
	npcsMu.Lock()
	defer npcsMu.Unlock()

	for _, spawn := range npcs {
		if strings.EqualFold(spawn.NPCID, input) || strings.EqualFold(spawn.NPCName, input) {
			return spawn, true
		}
	}

	return nil, false
}

// getDialogue returns a random dialogue line, or a default if none are set.
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

		npc, found := resolveNPC(step.Target)

		if found && npc == n {
			pq.StepIndex++
			complete := pq.StepIndex >= len(pq.Quest.Steps)

			player.PlayerMu.Unlock()

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

// handleTalk sends the target NPC's dialogue to the player.
func handleTalk(target string, player *Player) (string, error) {
	pname := player.getPlayerName()
	currLoc := player.getZoneID()

	currZone, ok := getZoneObj(currLoc)

	if !ok {
		return currLoc, InternalErr
	}

	spawn, ok := currZone.getNPCObject(target)

	if !ok {
		return currLoc, NPCNotFoundErr
	}

	pq, dialogue, completedQuest := spawn.getDialogue(player)
	sname := spawn.getNPCName()

			_, _ = fmt.Fprintln(player.Conn, "OK "+dialogue)
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+dialogue, "command", "TALK", "NPC", sname, "loc", currLoc)

		if completedQuest {
		pq.completeQuest(player)
	}

	return "", nil
}
