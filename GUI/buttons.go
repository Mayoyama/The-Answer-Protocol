package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"fyne.io/fyne/v2/widget"
)

func Capitalize(s string) string {
	if s == "" {
		return ""
	}

	firstChar, size := utf8.DecodeRuneInString(s)

	capitalized := string(unicode.ToUpper(firstChar)) + strings.ToLower(s[size:])

	return capitalized
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
	dirString := Capitalize(direction)
	dirButton := widget.NewButton(dirString, func() {
		select {
		case requests <- "MOVE " + direction:
		default:
		}
	})

	return dirButton
}
