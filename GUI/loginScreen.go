package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"strings"
	"unicode/utf8"
)

type LoginScreen struct {
	Content      *fyne.Container
	ConnErr      *widget.Label
	ConnButton   *widget.Button
	LoginContent *widget.Label
}

func buildLoginScreen(loginRequests chan<- string) *LoginScreen {
	ls := &LoginScreen{}

	ls.LoginContent = widget.NewLabel("Connect to TAP Server")

	ls.ConnErr = widget.NewLabel("")
	ls.ConnErr.Alignment = fyne.TextAlignCenter
	ls.ConnErr.Wrapping = fyne.TextWrapWord

	nameInput := widget.NewEntry()
	nameInput.SetPlaceHolder("Username (2-10 Characters)")

	ls.ConnButton = widget.NewButton("Connect", nil)
	ls.ConnButton.OnTapped = func() {
		name := strings.TrimSpace(nameInput.Text)
		nameLen := utf8.RuneCountInString(name)

		if nameLen < 2 || nameLen > 10 {
			ls.ConnErr.SetText("Username must be 2-10 characters")

			return
		}

		ls.ConnButton.Disable()
		ls.ConnErr.SetText("Connecting...")

		loginRequests <- name
	}

	ls.Content = container.NewVBox(
		ls.LoginContent,
		nameInput,
		ls.ConnButton,
		ls.ConnErr,
	)

	return ls
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

		cr.PlayerUsername = name

	} else {
		cr.PendingName = name
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

	cr.resetServerState()
}

func (ls *LoginScreen) showErr(message string) {
	fyne.Do(func() {
		ls.ConnErr.SetText(message)
		ls.ConnButton.Enable()
	})
}
