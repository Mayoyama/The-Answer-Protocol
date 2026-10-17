package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// createDirButtonSet builds the six MOVE buttons, keyed by direction.
func createDirButtonSet(requests chan<- string) map[string]*widget.Button {
	newSet := make(map[string]*widget.Button)
	dirSet := []string{"north", "south", "east", "west", "up", "down"}

	for _, d := range dirSet {
		dButton := createDirectionButton(d, requests)
		newSet[d] = dButton
	}

	return newSet
}

// createDirectionButton builds one MOVE button, labelled with the direction's letter or arrow.
func createDirectionButton(direction string, requests chan<- string) *widget.Button {
	var dirString string

	switch direction {
	case "north":
		dirString = "N"
	case "south":
		dirString = "S"
	case "east":
		dirString = "E"
	case "west":
		dirString = "W"
	case "up":
		dirString = "↑"
	case "down":
		dirString = "↓"
	default:
		dirString = "?"
	}

	dirButton := widget.NewButton(dirString, func() {
		select {
		case requests <- "MOVE " + direction:
		default:
		}
	})

	return dirButton
}

// createStatusGoldButton builds the STATUS button: it toggles the status/gold panel and
// sends STATUS and GOLD when opening it.
func (oTS *OverlayTopSection) createStatusGoldButton(tapConnRequests chan<- string) *widget.Button {
	newButton := widget.NewButton("STATUS", func() {
		if oTS.StatsGoldStack.Visible() {
			oTS.StatsGoldStack.Hide()

		} else {
			oTS.StatsGoldStack.Show()

			select {
			case tapConnRequests <- "STATUS":
			default:
			}

			select {
			case tapConnRequests <- "GOLD":
			default:
			}
		}
	})

	return newButton
}

// createRoomDetailsButton builds the LOOK button: it toggles the room details panel and
// sends LOOK when opening it.
func (oTS *OverlayTopSection) createRoomDetailsButton(tapConnRequests chan<- string) *widget.Button {
	newButton := widget.NewButton("LOOK", func() {
		if oTS.ZoneLookContent.Visible() {
			oTS.ZoneLookContent.Hide()

		} else {
			oTS.BattleLogContent.Hide()
			oTS.ZoneLookContent.Show()

			select {
			case tapConnRequests <- "LOOK":
			default:
			}
		}

		oTS.BattleSwapStack.Refresh()
	})

	return newButton
}

// createBattleLogButton builds the Battle Log button, which toggles the battle log panel.
func (oTS *OverlayTopSection) createBattleLogButton() *widget.Button {
	newButton := widget.NewButton("Battle Log", func() {
		if oTS.BattleLogContent.Visible() {
			oTS.BattleLogContent.Hide()

		} else {
			oTS.ZoneLookContent.Hide()
			oTS.BattleLogContent.Show()
		}

		oTS.BattleSwapStack.Refresh()
		oTS.BattleLog.Scroll.ScrollToBottom()
	})

	return newButton
}

// createStatusButton builds a button that sends cmd to the server when pressed.
func createStatusButton(label, cmd string, tapConnRequests chan<- string) *widget.Button {
	newButton := widget.NewButton(label, func() {
		select {
		case tapConnRequests <- cmd:
		default:
		}
	})

	return newButton
}

// createCmdPrefillButton builds a button that fills the command entry with prefillText.
func createCmdPrefillButton(label, prefillText string, textEntryWidget *widget.Entry, textCanvas fyne.Canvas) *widget.Button {
	newButton := widget.NewButton(label, func() {
		prefillEntry(prefillText, textEntryWidget, textCanvas)
	})

	return newButton
}

// prefillEntry fills the command entry with prefillText, moves the cursor to the end and
// focuses the entry.
func prefillEntry(prefillText string, textEntryWidget *widget.Entry, textCanvas fyne.Canvas) {
	textEntryWidget.SetText(prefillText)
	textEntryWidget.CursorColumn = len(prefillText)
	textEntryWidget.Refresh()
	textCanvas.Focus(textEntryWidget)
}

// createShowHideButton builds the button that hides and shows the overlay layers. Each
// element inside them keeps its own visibility.
func createShowHideButton(overlayWidgets []fyne.CanvasObject) *widget.Button {
	uiHidden := false

	var newButton *widget.Button

	newButton = widget.NewButton("🖼️", func() {
		if uiHidden {
			uiHidden = false

			for _, object := range overlayWidgets {
				object.Show()
			}

			newButton.SetText("🖼️")

		} else {
			uiHidden = true

			for _, object := range overlayWidgets {
				object.Hide()
			}

			newButton.SetText("👀")
		}
	})

	return newButton
}

// createPopUpMenuButton builds a button that opens a menu of the IDs in menuListContent
// (read when clicked); picking one sends "cmd <id>".
func createPopUpMenuButton(label, cmd string, menuListContent *[]string, tapConnRequests chan<- string, winCanvas fyne.Canvas) *widget.Button {
	var newButton *widget.Button

	newButton = widget.NewButton(label, func() {
		var menuItems []*fyne.MenuItem

		for _, id := range *menuListContent {
			item := fyne.NewMenuItem(IDToLabel(id), func() {
				select {
				case tapConnRequests <- cmd + " " + id:
				default:
				}
			})

			menuItems = append(menuItems, item)
		}

		if len(menuItems) == 0 {
			menuItems = append(menuItems, fyne.NewMenuItem("No Valid Target", func() {}))
		}

		menu := fyne.NewMenu("", menuItems...)
		below := fyne.NewPos(0, newButton.Size().Height)
		widget.ShowPopUpMenuAtRelativePosition(menu, winCanvas, below, newButton)
	})

	return newButton
}

// createExaminePopUpButton builds the EXAMINE button. Its menu fills the command entry
// with EXAMINE NPC, ITEM or KEYITEM; it sends nothing to the server itself.
func createExaminePopUpButton(textEntryWidget *widget.Entry, textCanvas fyne.Canvas) *widget.Button {
	var newButton *widget.Button

	newButton = widget.NewButton("🔍", func() {
		npcItem := fyne.NewMenuItem("NPC", func() {
			prefillEntry("EXAMINE NPC ", textEntryWidget, textCanvas)
		})

		itemItem := fyne.NewMenuItem("ITEM", func() {
			prefillEntry("EXAMINE ITEM ", textEntryWidget, textCanvas)
		})

		kiItem := fyne.NewMenuItem("KEY ITEM", func() {
			prefillEntry("EXAMINE KEYITEM ", textEntryWidget, textCanvas)
		})

		menu := fyne.NewMenu("", npcItem, itemItem, kiItem)
		popUp := widget.NewPopUpMenu(menu, textCanvas)
		above := fyne.NewPos(0, -popUp.MinSize().Height)
		popUp.ShowAtRelativePosition(above, newButton)
	})

	return newButton
}
