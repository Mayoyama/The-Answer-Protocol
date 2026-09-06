package main

import "fmt"

func EvtZoneEnter(username string) string {
	return fmt.Sprintf("EVT ROOM PRESENCE ENTER %s", username)
}

func EvtZoneLeave(username string) string {
	return fmt.Sprintf("EVT ROOM PRESENCE LEAVE %s", username)
}

func EvtZoneChat(username, message string) string {
	return fmt.Sprintf("EVT ROOM CHAT %s %s", username, message)
}

func EvtGlobalChat(username, message string) string {
	return fmt.Sprintf("EVT GLOBAL CHAT %s %s", username, message)
}

func EvtPartyInvite(leader string) string {
	return fmt.Sprintf("EVT GROUP INVITE %s", leader)
}

func EvtPartyJoin(username string) string {
	return fmt.Sprintf("EVT GROUP JOIN %s", username)
}

func EvtPartyLeave(username string) string {
	return fmt.Sprintf("EVT GROUP LEAVE %s", username)
}

func EvtPartyChat(username, message string) string {
	return fmt.Sprintf("EVT GROUP CHAT %s %s", username, message)
}

func EvtPlayerCount(count int) string {
	return fmt.Sprintf("EVT STATS players=%d", count)
}