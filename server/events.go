package main

import "fmt"

// EvtZoneEnter formats a room-presence-enter event for the given player.
func EvtZoneEnter(username string) string {
	return fmt.Sprintf("EVT ROOM PRESENCE ENTER %s", username)
}

// EvtZoneLeave formats a room-presence-leave event for the given player.
func EvtZoneLeave(username string) string {
	return fmt.Sprintf("EVT ROOM PRESENCE LEAVE %s", username)
}

// EvtZoneChat formats a room chat event from the given player.
func EvtZoneChat(username, message string) string {
	return fmt.Sprintf("EVT ROOM CHAT %s %s", username, message)
}

// EvtGlobalChat formats a global chat event from the given player.
func EvtGlobalChat(username, message string) string {
	return fmt.Sprintf("EVT GLOBAL CHAT %s %s", username, message)
}

// EvtPartyInvite formats a group-invite event from the given leader.
func EvtPartyInvite(leader string) string {
	return fmt.Sprintf("EVT GROUP INVITE %s", leader)
}

// EvtPartyJoin formats a group-join event for the given player.
func EvtPartyJoin(username string) string {
	return fmt.Sprintf("EVT GROUP JOIN %s", username)
}

// EvtPartyLeave formats a group-leave event for the given player.
func EvtPartyLeave(username string) string {
	return fmt.Sprintf("EVT GROUP LEAVE %s", username)
}

// EvtPartyChat formats a group chat event from the given player.
func EvtPartyChat(username, message string) string {
	return fmt.Sprintf("EVT GROUP CHAT %s %s", username, message)
}

// EvtPlayerCount formats a server-wide player count event.
func EvtPlayerCount(count int) string {
	return fmt.Sprintf("EVT STATS players=%d", count)
}
