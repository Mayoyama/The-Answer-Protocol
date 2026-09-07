package main

import (
	"sync"
)

var (
	zones = make(map[string]*Zone)
	zonesMu sync.Mutex
)

type Zone struct {
	ID string
	Name string
	Description string
	Exits map[string]string
	InZone map[string]*Player
	Items map[string]*Item
	NPCs map[string]*NPC
	ZoneMu sync.Mutex
}