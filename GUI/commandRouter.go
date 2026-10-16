package main

import (
	"fyne.io/fyne/v2"
	"strings"
	"time"
)

type CommandRouter struct {
	Win   fyne.Window
	Login *LoginScreen
	Game  *GameScreen

	LoginRequests   chan string
	TapConnRequests chan string
	QuitRequest     chan struct{}
	Quitting        bool

	tc              *tapConnection
	ServerResponses chan string
	ServerErrs      chan error
	GotGreeting     bool
	LoggedIn        bool
	PendingName     string

	PlayerUsername       string
	TalkTargetName       string
	CurrentZone          string
	InZoneCount          int
	PollTicker           *time.Ticker
	QueryWaitingForReply string

	AttackTargetName     string
	LastBattleTargetName string
	InBattle             bool
	FightOpen            bool
	LastAttack           time.Time
	BattleTimer          *time.Timer
	BattleLogHideTimer   *time.Timer
	battleScreenTimeout  chan struct{}
	battleLogHide        chan struct{}
}

func (cr *CommandRouter) run() {
	for {
		var pollChan <-chan time.Time

		if cr.PollTicker != nil {
			pollChan = cr.PollTicker.C
		}

		select {
		case name := <-cr.LoginRequests:
			cr.handleLoginRequest(name)

		case response := <-cr.ServerResponses:
			cr.handleServerResponse(response)

		case cmd := <-cr.TapConnRequests:
			cr.handleTapConnRequest(cmd)

		case err := <-cr.ServerErrs:
			cr.handleServerErr(err)

		case <-cr.QuitRequest:
			cr.handleQuit()

			return

		case <-pollChan:
			cr.handlePoll()

		case <-cr.battleScreenTimeout:
			cr.handleBattleScreenTimeout()

		case <-cr.battleLogHide:
			cr.handleBattleLogHide()
		}
	}
}

func (cr *CommandRouter) handleServerResponse(response string) {
	trimmedRes := strings.TrimSpace(response)

	if !cr.LoggedIn {
		if strings.HasPrefix(trimmedRes, "OK hello proto") {
			cr.GotGreeting = true

			if cr.PendingName != "" {
				if err := cr.tc.sendButtonCommandToServer("CONNECT " + cr.PendingName); err != nil {
					cr.Login.showErr(err.Error())
				}

				cr.PlayerUsername = cr.PendingName
			}

			cr.PendingName = ""

		} else if trimmedRes == "OK connected" {
			cr.LoggedIn = true
			cr.startPoll()

			if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
				cr.Game.displayGameError(err.Error())
			}

			if err := cr.tc.sendButtonCommandToServer("STATUS"); err != nil {
				cr.Game.displayGameError(err.Error())
			}

			if err := cr.tc.sendButtonCommandToServer("GOLD"); err != nil {
				cr.Game.displayGameError(err.Error())
			}

			fyne.Do(func() {
				cr.Win.SetContent(cr.Game.GameContent)
				cr.Win.Resize(fyne.NewSize(backdrop_w, backdrop_h+element_padding_h))
			})

		} else if strings.HasPrefix(trimmedRes, "ERR") {
			cr.Login.showErr(trimmedRes)
		}

	} else {
		if cr.Quitting && trimmedRes == "OK bye" {
			cr.resetServerState()

			cr.Game.displayGameError("OK bye")
			time.Sleep(time.Millisecond * 2800)

			fyne.Do(func() {
				cr.Win.Close()
			})

			return
		}

		reply, err := ParseServerReply(response)

		if err != nil {
			cr.Game.displayGameError(err.Error())

			return
		}

		switch reply.Type {
		case ReplyAttack:
			cr.handleBattle(reply)

		case ReplyBattleEnd:
			cr.logFlee(reply)
			cr.endBattleMode()

		case ReplyChat:
			cr.handleChat(reply)

		case ReplyEmptyList:
			cr.resolveEmptyList()

		case ReplyError:
			cr.handleReplyError(reply)

		case ReplyExamine:
			cr.handleExamine(reply)

		case ReplyGold:
			cr.handleGold(reply)

		case ReplyGroup:

		case ReplyInventory:
			cr.handleInventory(reply)

		case ReplyKeyitems:
			cr.handleKeyItems(reply)

		case ReplyLook:
			cr.handleLook(reply)

		case ReplyNPCDialogue:
			cr.handleNPCDialogue(reply)

		case ReplyPresence:
			cr.handlePresence(reply)

		case ReplyQuest:
			cr.handleQuest(reply)

		case ReplyQuests:
			cr.handleQuests(reply)

		case ReplyRespawned:
			cr.handleRespawn(reply)
			cr.endBattleMode()

		case ReplyReward:
			cr.handleRewards(reply)

		case ReplyRoomMove:
			cr.handleMove(reply)
			cr.logFlee(reply)
			cr.endBattleMode()

		case ReplyServerCount:
			cr.handleServerCount(reply)

		case ReplyStatus:
			cr.handleStatus(reply)

		case ReplyTakeDrop:
			cr.handleTakeDrop(reply)

		case ReplyWho:
			cr.handleWho(reply)

		case ReplyOther:
			fyne.Do(func() {
				cr.Game.OverlayBottomSection.LogWindow.addLine(reply.Other)
			})

		case ReplyIgnore:
		default:
		}
	}
}

func (cr *CommandRouter) handleTapConnRequest(cmd string) {
	if cr.tc == nil {
		cr.Game.displayGameError("Not Connected")

		return
	}

	upperCmd := strings.ToUpper(cmd)

	switch upperCmd {
	case "KEYITEMS", "INVENTORY", "QUESTS":
		cr.QueryWaitingForReply = upperCmd

	case "QUIT":
		cr.Quitting = true
	}

	cmdParts := strings.SplitN(cmd, " ", 2)

	if len(cmdParts) > 1 && strings.EqualFold(cmdParts[0], "attack") {
		cr.AttackTargetName = strings.TrimSpace(cmdParts[1])
	}

	if len(cmdParts) > 1 && strings.EqualFold(cmdParts[0], "talk") {
		cr.TalkTargetName = strings.TrimSpace(cmdParts[1])
	}

	if err := cr.tc.sendButtonCommandToServer(cmd); err != nil {
		cr.Game.displayGameError(err.Error())
	}
}

func (cr *CommandRouter) resetServerState() {
	cr.setBattleMode(false)
	cr.tc.shutdown()
	cr.stopPoll()

	cr.tc = nil
	cr.ServerResponses = nil
	cr.ServerErrs = nil
	cr.LastAttack = time.Time{}
	cr.GotGreeting = false
	cr.LoggedIn = false
	cr.Quitting = false
	cr.FightOpen = false
	cr.PendingName = ""
	cr.CurrentZone = ""
	cr.PlayerUsername = ""
	cr.AttackTargetName = ""
	cr.LastBattleTargetName = ""
	cr.TalkTargetName = ""
	cr.QueryWaitingForReply = ""

	oTS := cr.Game.OverlayTopSection
	oRS := cr.Game.OverlayRightSection
	oBS := cr.Game.OverlayBottomSection

	fyne.Do(func() {
		oTS.BattleLog.TextWin.Segments = nil
		oTS.BattleLog.TextWin.Refresh()

		oBS.ChatWindow.TextWin.Segments = nil
		oBS.ChatWindow.TextWin.Refresh()

		oBS.LogWindow.TextWin.Segments = nil
		oBS.LogWindow.TextWin.Refresh()

		oTS.BattleLogContent.Hide()
		oTS.ZoneLookContent.Show()
		oTS.BattleSwapStack.Refresh()

		cr.Game.BottomSection.NPCTargetIDs = nil

		oRS.InventoryList.ItemIDs = nil
		oRS.InventoryList.List.Refresh()
		oRS.TakeList.ItemIDs = nil
		oRS.TakeList.List.Refresh()
	})
}
