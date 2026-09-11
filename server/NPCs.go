package main

import (
	// "fmt"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
)

type NPCRole int

const (
	General NPCRole = iota
	QuestGiver
	Enemy
)

var (
	npcs   = make(map[string]*NPC)
	npcsMu sync.Mutex
)

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

func (n *NPC) getDialogue() string {
	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()
	chats := len(n.Dialogue)

	if chats == 0 {
		return "..." //default dialogue if no dialogue is set
	}

	option := rand.Intn(chats)

	return n.Dialogue[option]
}

func handleTalk(target string, player *Player) (error, string) {
	pname := player.getPlayerName()
	currLoc := player.getZoneID()
	currZone, ok := getZoneObj(currLoc)
	if !ok {
		return InternalErr, currLoc
	}

	spawn, ok := currZone.getNPCObject(target)
	if !ok {
		return NPCNotFoundErr, currLoc
	}

	dialogue := spawn.getDialogue()
	sname := spawn.getNPCName()

	fmt.Fprintln(player.Conn, "OK "+dialogue)
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+dialogue, "command", "TALK", "NPC", sname, "loc", currLoc)

	return nil, ""
}
