package main

import (
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
