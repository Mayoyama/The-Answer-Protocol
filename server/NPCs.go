package main

import (
	// "fmt"
	"sync"
)

type NPCRole int

const (
	General NPCRole = iota
	QuestGiver
	Enemy
)

var (
	npcs  = make(map[string]*NPC)
	npcMu sync.Mutex
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
}
