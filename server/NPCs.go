package main

import (
	// "fmt"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
)

// NPCRole categorizes an NPC's behavior (general, quest giver, enemy).
type NPCRole int

// NPC role values.
const (
	General NPCRole = iota
	QuestGiver
	Enemy
)

// npcs holds all loaded NPCs, keyed by NPC ID.
var (
	npcs   = make(map[string]*NPC)
	npcsMu sync.Mutex
)

// NPC represents a non-player character in the world.
type NPC struct {
	NPCID       string
	NPCName     string
	Description string
	Dialogue    []string
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
func (n *NPC) getDialogue() string {
	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()
	chats := len(n.Dialogue)

	if chats == 0 {
		return "..." //default dialogue if no dialogue is set
	}

	option := rand.IntN(chats)

	return n.Dialogue[option]
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

	dialogue := spawn.getDialogue()
	sname := spawn.getNPCName()

	_, _ = fmt.Fprintln(player.Conn, "OK "+dialogue)
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+dialogue, "command", "TALK", "NPC", sname, "loc", currLoc)

	return "", nil
}
