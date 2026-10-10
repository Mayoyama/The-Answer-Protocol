package main

import (
	"fmt"
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

	tc              *tapConnection
	ServerResponses chan string
	ServerErrs      chan error
	GotGreeting     bool
	LoggedIn        bool
	PendingName     string
	CurrentZone     string
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

			fyne.Do(func() {
				cr.Win.SetContent(cr.Game.GameContent)
				cr.Win.Resize(fyne.NewSize(backdrop_w, backdrop_h+element_padding_h))
			})

		} else if strings.HasPrefix(trimmedRes, "ERR") {
			cr.Login.showErr(trimmedRes)
		}

	} else {
		reply, err := ParseServerReply(response)

		if err != nil {
			cr.Game.displayGameError(err.Error())

			return
		}

		switch reply.Type {
		case ReplyAttack:
		case ReplyBattleEnd:
		case ReplyChat:
		case ReplyError:
			cr.handleReplyError(reply)

		case ReplyExamine:
		case ReplyGold:
		case ReplyGroup:
		case ReplyIgnore:
		case ReplyInventory:
		case ReplyKeyitems:
		case ReplyLook:
			roomText := fmt.Sprintf("%s\n%s", reply.Look.Room.Name, reply.Look.Room.Description)

			fyne.Do(func() {
				cr.Game.ZoneNameDescLabel.SetText(roomText)
			})

			if reply.Look.Room.ID != cr.CurrentZone {
				cr.CurrentZone = reply.Look.Room.ID
				cr.Game.GameScreen.showZoneImage(reply.Look.Room.ID)
			}

		case ReplyPresence:
		case ReplyQuest:
		case ReplyQuests:
		case ReplyRespawned:
			if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
				cr.Game.displayGameError(err.Error())
			}

		case ReplyRoomMove:
			cr.CurrentZone = reply.RoomID

			cr.Game.GameScreen.showZoneImage(reply.RoomID)

			if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
				cr.Game.displayGameError(err.Error())
			}

		case ReplyServerCount:
		case ReplyStatus:
		case ReplyTakeDrop:
		case ReplyWho:
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
