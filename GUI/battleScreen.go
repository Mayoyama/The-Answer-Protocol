package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
)

const battleIdleTimeout time.Duration = time.Second * 10

func (cr *CommandRouter) setBattleMode(on bool) {
	if on == cr.InBattle {
		return
	}

	cr.InBattle = on

	if !on && cr.BattleTimer != nil {
		cr.BattleTimer.Stop()
	}

	oTS := cr.Game.OverlayTopSection

	fyne.Do(func() {
		if on {
			oTS.BoxWasVisible = oTS.ZoneLookContent.Visible()
			oTS.ZoneLookContent.Hide()
			oTS.BattleLogContent.Show()

		} else {
			oTS.BattleLogContent.Hide()

			if oTS.BoxWasVisible {
				oTS.ZoneLookContent.Show()
			}
		}

		oTS.BattleSwapStack.Refresh()
	})
}

func (cr *CommandRouter) restartBattleTimer() {
	cr.LastAttack = time.Now()

	if cr.BattleTimer != nil {
		cr.BattleTimer.Stop()
	}

	timeoutChan := cr.battleScreenTimeout

	cr.BattleTimer = time.AfterFunc(battleIdleTimeout, func() {
		select {
		case timeoutChan <- struct{}{}:
		default:
		}
	})
}

func (cr *CommandRouter) handleBattleScreenTimeout() {
	if time.Since(cr.LastAttack) < battleIdleTimeout {
		return
	}

	cr.setBattleMode(false)
}

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
		cr.setBattleMode(false)
	}
}

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
		targetName = "Enemy"
	}

	playerFirst = data.PlayerInitRoll >= data.TargetInitRoll

	if playerFirst {
		whoFirst = cr.PlayerUsername

	} else {
		whoFirst = targetName
	}

	initString := fmt.Sprintf("Battle Round! Initiative: %s rolls %d, %s rolls %d. %s goes first.", cr.PlayerUsername, data.PlayerInitRoll, targetName, data.TargetInitRoll, whoFirst)

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
