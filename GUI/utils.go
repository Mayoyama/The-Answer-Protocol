package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func Capitalize(s string) string {
	if s == "" {
		return ""
	}

	firstChar, size := utf8.DecodeRuneInString(s)

	capitalized := string(unicode.ToUpper(firstChar)) + strings.ToLower(s[size:])

	return capitalized
}

func SplitSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var noSnake string

	words := strings.Split(s, "_")

	for i, w := range words {
		if i > 0 {
			noSnake += " "
		}
		noSnake += Capitalize(w)
	}

	return noSnake
}

func IDToLabel(id string) string {
	if id == "" {
		return ""
	}

	name := id

	name = strings.TrimPrefix(name, "item.")
	name = strings.TrimPrefix(name, "key_item.")
	name = strings.TrimPrefix(name, "npc.")
	name = strings.TrimPrefix(name, "quest.")
	name = strings.TrimPrefix(name, "zone.")
	name = strings.TrimPrefix(name, "room.")
	name = strings.TrimPrefix(name, "loc.")

	return SplitSnakeCase(name)
}

func buildQuestLine(q QuestsParse) string {
	qLine := fmt.Sprintf("%s (%s)", IDToLabel(q.QuestID), Capitalize(q.Status))

	if q.Progress != "" {
		qLine += fmt.Sprintf(" - Steps: %s", q.Progress)
	}

	if q.QuestItems != "" {
		qLine += fmt.Sprintf((" - Items: %s"), q.QuestItems)
	}

	return qLine
}
