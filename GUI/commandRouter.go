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
	CurrentZone     string
	InZoneCount     int
}

func (cr *CommandRouter) run() {
	for {
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
		}
	}
}

func (cr *CommandRouter) handleLoginRequest(name string) {
	if cr.tc == nil {
		newConn, err := dialServer()

		if err != nil {
			cr.Login.showErr(err.Error())

			return
		}

		cr.tc = newConn
		cr.ServerResponses = cr.tc.ServerResponses
		cr.ServerErrs = cr.tc.ServerErrs
	}

	if cr.GotGreeting {
		if err := cr.tc.sendButtonCommandToServer("CONNECT " + name); err != nil {
			cr.Login.showErr(err.Error())
		}

	} else {
		cr.PendingName = name
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
			}

			cr.PendingName = ""

		} else if trimmedRes == "OK connected" {
			cr.LoggedIn = true

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
			cr.tc.shutdown()

			cr.tc = nil
			cr.ServerResponses = nil
			cr.ServerErrs = nil
			cr.GotGreeting = false
			cr.LoggedIn = false
			cr.Quitting = false
			cr.PendingName = ""
			cr.CurrentZone = ""

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
		case ReplyBattleEnd:
		case ReplyChat:
			cr.handleChat(reply)
		case ReplyError:
			cr.handleReplyError(reply)

		case ReplyExamine:
			cr.handleExamine(reply)
		case ReplyGold:
			cr.handleGold(reply)
		case ReplyGroup:
		case ReplyIgnore:
		case ReplyInventory:
		case ReplyKeyitems:
		case ReplyLook:
			cr.handleLook(reply)

		case ReplyNPCDialogue:
			cr.handleNPCDialogue(reply)

		case ReplyPresence:
			//cr.handlePresence()
		case ReplyQuest:
		case ReplyQuests:
		case ReplyRespawned:
			cr.handleRespawn()

		case ReplyRoomMove:
			cr.handleMove(reply)

		case ReplyServerCount:
			cr.handleServerCount(reply)
		case ReplyStatus:
			cr.handleStatus(reply)
		case ReplyTakeDrop:
		case ReplyWho:
			cr.handleWho(reply)
		case ReplyOther:
		default:
		}
	}
}

func (cr *CommandRouter) handleTapConnRequest(cmd string) {
	if cr.tc == nil {
		cr.Game.displayGameError("Not Connected")

		return
	}

	if strings.ToUpper(cmd) == "QUIT" {
		cr.Quitting = true
	}

	if err := cr.tc.sendButtonCommandToServer(cmd); err != nil {
		cr.Game.displayGameError(err.Error())
	}
}

func (cr *CommandRouter) handleServerErr(err error) {
	fyne.Do(func() {
		cr.Login.LoginContent.SetText("Connect to TAP Server")
		cr.Login.ConnErr.SetText("Connection lost: " + err.Error())
		cr.Login.ConnButton.Enable()
		cr.Win.SetContent(cr.Login.Content)
		cr.Win.Resize(fyne.NewSize(login_win_w, login_win_h))
	})

	cr.tc.shutdown()

	cr.tc = nil
	cr.ServerResponses = nil
	cr.ServerErrs = nil
	cr.GotGreeting = false
	cr.LoggedIn = false
	cr.Quitting = false
	cr.PendingName = ""
	cr.CurrentZone = ""
}

func (cr *CommandRouter) handleQuit() {
	if cr.tc != nil {
		var quitMsg string

		quitErr := cr.tc.gracefulQuit(time.Second * 3)
		cr.tc.shutdown()

		if quitErr != nil {
			quitMsg = quitErr.Error()

		} else {
			quitMsg = "OK bye"
		}

		if cr.LoggedIn {
			cr.Game.displayGameError(quitMsg)

		} else {
			fyne.Do(func() {
				cr.Login.ConnErr.SetText(quitMsg)

			})
		}

		time.Sleep(time.Millisecond * 2800)
	}

	fyne.Do(func() {
		cr.Win.Close()
	})
}
