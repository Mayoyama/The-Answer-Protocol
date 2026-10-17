package main

import "time"

// pollInterval is how often LOOK is polled while other players are in the room.
const pollInterval time.Duration = time.Millisecond * 3500

// startPoll starts, or restarts, the LOOK poll ticker.
func (cr *CommandRouter) startPoll() {
	cr.stopPoll()
	cr.PollTicker = time.NewTicker(pollInterval)
}

// stopPoll stops the LOOK poll ticker.
func (cr *CommandRouter) stopPoll() {
	if cr.PollTicker != nil {
		cr.PollTicker.Stop()
		cr.PollTicker = nil
	}
}

// handlePoll sends LOOK on each tick while logged in and sharing the room with another player.
func (cr *CommandRouter) handlePoll() {
	if cr.tc == nil || !cr.LoggedIn {
		return
	}

	if cr.InZoneCount <= 1 {
		return
	}

	_ = cr.tc.sendButtonCommandToServer("LOOK")
}
