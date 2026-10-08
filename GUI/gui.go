package main

import (
	"strings"
	"time"
	"encoding/json"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	guiApp := app.New()
	win := guiApp.NewWindow("The Answer Protocol")

	loginRequests := make(chan string, 1)
	quitRequest := make(chan struct{}, 1)

	win.SetCloseIntercept(func() {
		select {
		case quitRequest <- struct{}{}:
		default:

		}

	})

	connect := widget.NewLabel("Connect to TAP Server")

	connErr := widget.NewLabel("")
	connErr.Alignment = fyne.TextAlignCenter
	connErr.Wrapping = fyne.TextWrapWord

	nameInput := widget.NewEntry()
	nameInput.SetPlaceHolder("Username (2-10 Characters)")

	connButton := widget.NewButton("Connect", nil)
	connButton.OnTapped = func() {
		name := nameInput.Text

		connButton.Disable()

		connErr.SetText("Connecting...")

		loginRequests <- name
	}

	win.SetContent(container.NewVBox(
		connect,
		nameInput,
		connButton,
		connErr,
	))

	win.Resize(fyne.NewSize(400, 200))

	go func() {
		var (
			tc              *tapConnection
			serverResponses chan string
			serverErrs      chan error
			gotGreeting     = false
			loggedIn        = false
			pendingName     string
		)

		for {
			select {
			case name := <-loginRequests:
				if tc == nil {
					newConn, err := dialServer()

					if err != nil {
						fyne.Do(func() {
							connErr.SetText(err.Error())
							connButton.Enable()
						})

						continue
					}

					tc = newConn
					serverResponses = tc.ServerResponses
					serverErrs = tc.ServerErrs
				}

				if gotGreeting {
					if err := tc.sendButtonCommandToServer("CONNECT " + name); err != nil {
						fyne.Do(func() {
							connErr.SetText(err.Error())
							connButton.Enable()
						})
					}
				} else {
					pendingName = name
				}

			case response := <-serverResponses:
				trimmedRes := strings.TrimSpace(response)

				if !loggedIn {
					if strings.HasPrefix(trimmedRes, "OK hello proto") {
						gotGreeting = true

						if pendingName != "" {
							if err := tc.sendButtonCommandToServer("CONNECT " + pendingName); err != nil {
								fyne.Do(func() {
									connErr.SetText(err.Error())
									connButton.Enable()
								})
							}
						}

						pendingName = ""

					} else if trimmedRes == "OK connected" {
						loggedIn = true

						fyne.Do(func() {
							connect.SetText("Connected")
							connErr.SetText("")
						})

					} else if strings.HasPrefix(trimmedRes, "ERR") {
						fyne.Do(func() {
							connErr.SetText(trimmedRes)
							connButton.Enable()
						})
					}
				}

			case err := <-serverErrs:
				fyne.Do(func() {
					connect.SetText("Connect to TAP Server")
					connErr.SetText("Connection lost: " + err.Error())
					connButton.Enable()
				})

				tc.shutdown()

				tc = nil
				serverResponses = nil
				serverErrs = nil
				gotGreeting = false
				loggedIn = false
				pendingName = ""

			case <-quitRequest:
				if tc != nil {
					_ = tc.gracefulQuit(time.Second * 3)
					tc.shutdown()
				}

				fyne.Do(func() {
					win.Close()
				})

				return
			}
		}
	}()

	win.ShowAndRun()
}

// case <-time.After(time.Second * 5):
// 	if err := tc.gracefulQuit(quitTimeoutDuration); err != nil {
// 		_, _ = fmt.Println(err)

// 	} else {
// 		_, _ = fmt.Println("OK bye")
// 	}
// 	return
