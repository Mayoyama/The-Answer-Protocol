package main

import (
	"fyne.io/fyne/v2/widget"
)

type ItemList struct {
	ItemIDs []string
	List    *widget.List
}

func createDirButtonSet(requests chan<- string) map[string]*widget.Button {
	newSet := make(map[string]*widget.Button)
	dirSet := []string{"north", "south", "east", "west", "up", "down"}

	for _, d := range dirSet {
		dButton := createDirectionButton(d, requests)
		newSet[d] = dButton
	}

	return newSet
}

func createDirectionButton(direction string, requests chan<- string) *widget.Button {
	var dirString string

	switch direction {
	case "north":
		dirString = "N"
	case "south":
		dirString = "S"
	case "east":
		dirString = "E"
	case "west":
		dirString = "W"
	case "up":
		dirString = "↑"
	case "down":
		dirString = "↓"
	default:
		dirString = "?"
	}

	dirButton := widget.NewButton(dirString, func() {
		select {
		case requests <- "MOVE " + direction:
		default:
		}
	})

	return dirButton
}

func createStatusButton(label, cmd string, cmdChan chan<- string) *widget.Button {
	newButton := widget.NewButton(label, func() {
		select {
		case cmdChan <- cmd:
		default:
		}
	})

	return newButton
}


