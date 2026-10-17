package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"strings"
	"unicode/utf8"
)

// LoginScreen holds the login screen's widgets.
type LoginScreen struct {
	Content      *fyne.Container
	ConnErr      *widget.Label
	ConnButton   *widget.Button
	LoginContent *widget.Label
}

// buildLoginScreen builds the login screen. Connect checks the name is 2–10 characters
// before passing it to the router.
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

// handleLoginRequest dials the server if needed and sends CONNECT, or keeps the name
// until the server's greeting arrives.
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

// handleServerErr returns to the login screen after a lost connection and resets the session.
func (cr *CommandRouter) handleServerErr(err error) {
	fyne.Do(func() {
		cr.Login.LoginContent.SetText("Connect to TAP Server")
		cr.Login.ConnErr.SetText("Connection lost: " + err.Error())
		cr.Login.ConnButton.Enable()
		cr.Win.SetContent(cr.Login.Content)
		cr.Win.Resize(fyne.NewSize(login_win_w, login_win_h))
		cr.Win.CenterOnScreen()
	})

	cr.resetServerState()
}

// showErr shows message under the Connect button and enables the button again.
func (ls *LoginScreen) showErr(message string) {
	fyne.Do(func() {
		ls.ConnErr.SetText(message)
		ls.ConnButton.Enable()
	})
}
