package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"strings"
	"time"
)

const (
	// BattleIdleTimeout is how long without an ATTACK reply before the GUI leaves fight mode.
	BattleIdleTimeout time.Duration = time.Second * 10
	// BattleLogHideDelay is how long the battle log stays visible after a fight ends.
	BattleLogHideDelay time.Duration = time.Second * 5
)

// setBattleMode turns fight mode on or off. Turning it on opens the battle log, adds a
// start divider when the opponent changed and enables FLEE; turning it off starts the
// battle log hide timer.
func (cr *CommandRouter) setBattleMode(on bool) {
	if on == cr.InBattle {
		return
	}

	cr.InBattle = on

	if !on && cr.BattleTimer != nil {
		cr.BattleTimer.Stop()
	}

	if on {
		if cr.BattleLogHideTimer != nil {
			cr.BattleLogHideTimer.Stop()
		}

		cr.FightOpen = true

		targetName := IDToLabel(cr.AttackTargetName)

		newTarget := !strings.EqualFold(targetName, cr.LastBattleTargetName)
		cr.LastBattleTargetName = targetName

		if targetName == "" {
			targetName = "Enemy NPC"
		}

		separator := fmt.Sprintf("=========== Battle: %s ===========", targetName)

		oTS := cr.Game.OverlayTopSection

		fyne.Do(func() {
			if newTarget {
				oTS.BattleLog.addLine(separator)
			}

			cr.Game.BottomSection.FleeButton.Enable()
			oTS.ZoneLookContent.Hide()
			oTS.BattleLogContent.Show()
			oTS.BattleSwapStack.Refresh()
			oTS.BattleLog.Scroll.ScrollToBottom()
		})

	} else {
		cr.startBattleLogHideTimer()
	}
}

// restartBattleTimer records the time of the latest ATTACK reply and restarts the idle
// timer, which signals battleScreenTimeout after BattleIdleTimeout.
func (cr *CommandRouter) restartBattleTimer() {
	cr.LastAttack = time.Now()

	if cr.BattleTimer != nil {
		cr.BattleTimer.Stop()
	}

	timeoutChan := cr.battleScreenTimeout

	cr.BattleTimer = time.AfterFunc(BattleIdleTimeout, func() {
		select {
		case timeoutChan <- struct{}{}:
		default:
		}
	})
}

// handleBattleScreenTimeout leaves fight mode if no ATTACK reply arrived within BattleIdleTimeout.
func (cr *CommandRouter) handleBattleScreenTimeout() {
	if time.Since(cr.LastAttack) < BattleIdleTimeout {
		return
	}

	cr.setBattleMode(false)
}

// startBattleLogHideTimer restarts the timer that signals battleLogHide after BattleLogHideDelay.
func (cr *CommandRouter) startBattleLogHideTimer() {
	if cr.BattleLogHideTimer != nil {
		cr.BattleLogHideTimer.Stop()
	}

	hideChan := cr.battleLogHide

	cr.BattleLogHideTimer = time.AfterFunc(BattleLogHideDelay, func() {
		select {
		case hideChan <- struct{}{}:
		default:
		}
	})
}

// handleBattle writes one ATTACK round to the battle log, updates the HP bar and ends
// the fight when either side reaches 0 HP.
func (cr *CommandRouter) handleBattle(reply ReplyContent) {
	atk := reply.Attack

	cr.setBattleMode(true)
	cr.restartBattleTimer()

	atkRound := cr.buildAttackRoundString(atk)

	fyne.Do(func() {
		cr.Game.OverlayTopSection.BattleLog.addLine(atkRound)
		cr.Game.TopSection.HPBar.Max = float64(atk.PlayerMaxHP)
		cr.Game.TopSection.HPBar.SetValue(float64(atk.PlayerHP))
	})

	if atk.PlayerHP <= 0 || atk.TargetHP <= 0 {
		cr.endBattleMode()
	}
}

// handleBattleLogHide hides the battle log unless a new fight has started.
func (cr *CommandRouter) handleBattleLogHide() {
	if cr.InBattle {
		return
	}

	oTS := cr.Game.OverlayTopSection

	fyne.Do(func() {
		oTS.BattleLogContent.Hide()
		oTS.BattleSwapStack.Refresh()
	})
}

// buildAttackRoundString formats one ATTACK reply as an initiative line followed by the
// hits and dodges in initiative order.
func (cr *CommandRouter) buildAttackRoundString(data AttackParse) string {
	var (
		battleString string
		whoFirst     string
		playerFirst  bool
		playerEffect string
		targetEffect string
		firstEffect  string
		secondEffect string
	)

	targetName := IDToLabel(cr.AttackTargetName)

	if targetName == "" {
		targetName = "Enemy NPC"
	}

	playerFirst = data.PlayerInitRoll >= data.TargetInitRoll

	if playerFirst {
		whoFirst = cr.PlayerUsername

	} else {
		whoFirst = targetName
	}

	initString := fmt.Sprintf("Battle Round! Initiative 🎲: %s rolls %d, %s rolls %d. %s goes first.", cr.PlayerUsername, data.PlayerInitRoll, targetName, data.TargetInitRoll, whoFirst)

	if data.PlayerHP <= 0 && data.TargetDmgRecvd == 0 {
		playerEffect = ""

	} else if data.TargetDodged {
		playerEffect = fmt.Sprintf("%s dodged your attack!", targetName)

	} else {
		playerEffect = fmt.Sprintf("You inflict %d points of damage onto %s. (%d/%d)", data.TargetDmgRecvd, targetName, data.TargetHP, data.TargetMaxHP)
	}

	if data.TargetHP <= 0 && data.PlayerDmgRecvd == 0 {
		targetEffect = ""

	} else if data.PlayerDodged {
		targetEffect = fmt.Sprintf("You dodged %s's attack!", targetName)

	} else {
		targetEffect = fmt.Sprintf("%s hits you for %d points of damage.", targetName, data.PlayerDmgRecvd)
	}

	if playerFirst {
		firstEffect = playerEffect
		secondEffect = targetEffect

	} else {
		firstEffect = targetEffect
		secondEffect = playerEffect
	}

	battleString = firstEffect

	if secondEffect != "" {
		if battleString != "" {
			battleString += " "
		}

		battleString += secondEffect
	}

	return fmt.Sprintf("%s\n%s", initString, battleString)
}

// logFlee adds the flee line to the battle log when the reply came from FLEE.
func (cr *CommandRouter) logFlee(reply ReplyContent) {
	if reply.BattleFled {
		fyne.Do(func() {
			cr.Game.OverlayTopSection.BattleLog.addLine("🏃 You fled the battle.")
		})
	}
}

// endBattleMode closes an open fight (end divider, FLEE disabled, battle log kept visible
// for BattleLogHideDelay) and leaves fight mode.
func (cr *CommandRouter) endBattleMode() {
	if cr.FightOpen {
		cr.FightOpen = false

		oTS := cr.Game.OverlayTopSection

		fyne.Do(func() {
			oTS.BattleLog.addLine("**********************************")
			oTS.ZoneLookContent.Hide()
			oTS.BattleLogContent.Show()
			oTS.BattleSwapStack.Refresh()
			oTS.BattleLog.Scroll.ScrollToBottom()
			cr.Game.BottomSection.FleeButton.Disable()
		})

		cr.startBattleLogHideTimer()
	}

	cr.setBattleMode(false)
}
