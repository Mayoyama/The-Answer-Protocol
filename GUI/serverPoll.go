package main

import "time"

const pollInterval time.Duration = time.Millisecond * 3500

func (cr *CommandRouter) startPoll() {
	cr.stopPoll()
	cr.PollTicker = time.NewTicker(pollInterval)
}

func (cr *CommandRouter) stopPoll() {
	if cr.PollTicker != nil {
		cr.PollTicker.Stop()
		cr.PollTicker = nil
	}
}

func (cr *CommandRouter) handlePoll() {
	if cr.tc == nil || !cr.LoggedIn {
		return
	}

	if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
		return
	}
}
