package main

import (
	"fyne.io/fyne/v2"
	"strings"
	"time"
)

func (cr *CommandRouter) handleReplyError(reply ReplyContent) {
	cr.Game.displayGameError(reply.Err.Text)

	if reply.Err.Code == 750 {
		lockDuration := time.Minute

		_, timeString, found := strings.Cut(reply.Err.Text, "Time remaining: ")

		if found {
			penaltyTime, parseErr := time.ParseDuration(strings.TrimSpace(timeString))

			if parseErr == nil {
				lockDuration = penaltyTime
			}
		}

		if cr.Game.LockoutTimer != nil {
			cr.Game.LockoutTimer.Stop()
		}

		fyne.Do(func() {
			for _, button := range cr.Game.DirButtonSet {
				button.Disable()
			}
		})

		cr.Game.LockoutTimer = time.AfterFunc(lockDuration, func() {
			fyne.Do(func() {
				for _, button := range cr.Game.DirButtonSet {
					button.Enable()
				}
			})
		})
	}
}
