package main

import (
	//"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	var (
		errTimer     *time.Timer
		lockoutTimer *time.Timer
	)

	guiApp := app.New()
	win := guiApp.NewWindow("The Answer Protocol")

	loginRequests := make(chan string, 1)
	tapConnRequests := make(chan string, 16)
	quitRequest := make(chan struct{}, 1)

	const (
		login_win_w       float32 = 400
		login_win_h       float32 = 200
		backdrop_w        float32 = 1280
		backdrop_h        float32 = 720
		element_padding_h float32 = 180
	)

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

	gameErr := widget.NewLabel("")
	gameErr.Wrapping = fyne.TextWrapWord

	showGameErr := func(errMessage string) {
		if errTimer != nil {
			errTimer.Stop()
		}

		fyne.Do(func() {
			gameErr.SetText(errMessage)
		})

		errTimer = time.AfterFunc(time.Second*3, func() {
			fyne.Do(func() {
				gameErr.SetText("")
			})
		})
	}

	nameInput := widget.NewEntry()
	nameInput.SetPlaceHolder("Username (2-10 Characters)")

	connButton := widget.NewButton("Connect", nil)
	connButton.OnTapped = func() {
		name := strings.TrimSpace(nameInput.Text)

		nameLen := utf8.RuneCountInString(name)

		if nameLen < 2 || nameLen > 10 {
			connErr.SetText("Username must be 2-10 characters")

			return
		}

		connButton.Disable()

		connErr.SetText("Connecting...")

		loginRequests <- name
	}

	loginContent := container.NewVBox(
		connect,
		nameInput,
		connButton,
		connErr,
	)

	win.SetContent(loginContent)
	win.Resize(fyne.NewSize(login_win_w, login_win_h))

	dirButtonSet := createDirButtonSet(tapConnRequests)

	dPad := container.NewGridWithColumns(
		3, widget.NewLabel(""), dirButtonSet["north"], dirButtonSet["up"],
		dirButtonSet["west"], widget.NewLabel(""), dirButtonSet["east"],
		widget.NewLabel(""), dirButtonSet["south"], dirButtonSet["down"])

	if errors := loadZoneImages(); len(errors) != 0 {
		_, _ = fmt.Println("Errors while loading zone images:")

		for i, err := range errors {
			_, _ = fmt.Printf("%d: %v\n", i, err)
		}

		//return
	}

	newScreen := buildNewZoneImage(backdrop_w, backdrop_h)

	zoneNameDescLabel := widget.NewLabel("")
	zoneNameDescLabel.Alignment = fyne.TextAlignCenter
	zoneNameDescLabel.Wrapping = fyne.TextWrapWord

	gameContent := container.NewVBox(newScreen.zoneStack, zoneNameDescLabel, dPad, gameErr)

	go func() {
		var (
			tc              *tapConnection
			serverResponses chan string
			serverErrs      chan error
			gotGreeting     = false
			loggedIn        = false
			pendingName     string
			currentZone     string
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

						if err := tc.sendButtonCommandToServer("LOOK"); err != nil {
							showGameErr(err.Error())
						}

						fyne.Do(func() {
							win.SetContent(gameContent)
							win.Resize(fyne.NewSize(backdrop_w, backdrop_h+element_padding_h))
						})

					} else if strings.HasPrefix(trimmedRes, "ERR") {
						fyne.Do(func() {
							connErr.SetText(trimmedRes)
							connButton.Enable()
						})
					}

				} else {
					reply, err := ParseServerReply(response)

					if err != nil {
						showGameErr(err.Error())

						continue
					}

					switch reply.Type {
					case ReplyAttack:
					case ReplyBattleEnd:
					case ReplyChat:
					case ReplyError:
						showGameErr(reply.Err.Text)

						if reply.Err.Code == 750 {
							lockDuration := time.Minute

							_, timeString, found := strings.Cut(reply.Err.Text, "Time remaining: ")

							if found {
								penaltyTime, parseErr := time.ParseDuration(strings.TrimSpace(timeString))

								if parseErr == nil {
									lockDuration = penaltyTime
								}
							}

							if lockoutTimer != nil {
								lockoutTimer.Stop()
							}

							fyne.Do(func() {
								for _, button := range dirButtonSet {
									button.Disable()
								}
							})

							lockoutTimer = time.AfterFunc(lockDuration, func() {
								fyne.Do(func() {
									for _, button := range dirButtonSet {
										button.Enable()
									}
								})
							})
						}

					case ReplyExamine:
					case ReplyGold:
					case ReplyGroup:
					case ReplyIgnore:
					case ReplyInventory:
					case ReplyKeyitems:
					case ReplyLook:
						roomText := fmt.Sprintf("%s\n%s", reply.Look.Room.Name, reply.Look.Room.Description)

						fyne.Do(func() {
							zoneNameDescLabel.SetText(roomText)
						})

						if reply.Look.Room.ID != currentZone {
							currentZone = reply.Look.Room.ID
							newScreen.showZoneImage(reply.Look.Room.ID)
						}

					case ReplyPresence:
					case ReplyQuest:
					case ReplyQuests:
					case ReplyRespawned:
						if err := tc.sendButtonCommandToServer("LOOK"); err != nil {
							showGameErr(err.Error())
						}

					case ReplyRoomMove:
						currentZone = reply.RoomID

						newScreen.showZoneImage(reply.RoomID)

						if err := tc.sendButtonCommandToServer("LOOK"); err != nil {
							showGameErr(err.Error())
						}

					case ReplyServerCount:
					case ReplyStatus:
					case ReplyTakeDrop:
					case ReplyWho:
					case ReplyOther:
					default:
					}
				}

			case cmd := <-tapConnRequests:
				if tc == nil {
					showGameErr("Not Connected")

					continue
				}

				if err := tc.sendButtonCommandToServer(cmd); err != nil {
					showGameErr(err.Error())
				}

			case err := <-serverErrs:
				fyne.Do(func() {
					connect.SetText("Connect to TAP Server")
					connErr.SetText("Connection lost: " + err.Error())
					connButton.Enable()
					win.SetContent(loginContent)
					win.Resize(fyne.NewSize(login_win_w, login_win_h))
				})

				tc.shutdown()

				tc = nil
				serverResponses = nil
				serverErrs = nil
				gotGreeting = false
				loggedIn = false
				pendingName = ""
				currentZone = ""

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
