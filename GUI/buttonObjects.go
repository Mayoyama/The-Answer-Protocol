package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ItemList struct {
	ItemIDs []string
	List    *widget.List
}

func createDirButtonSet(requests chan<- string) map[string]*widget.Button {
	newSet := make(map[string]*widget.Button)
	dirSet := []string{"north", "south", "east", "west", "up", "down"}

	for _, d := range dirSet {
		dButton := createDirectionButton(d, requests)
		newSet[d] = dButton
	}

	return newSet
}

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

func createStatusButton(label, cmd string, tapConnRequests chan<- string) *widget.Button {
	newButton := widget.NewButton(label, func() {
		select {
		case tapConnRequests <- cmd:
		default:
		}
	})

	return newButton
}

func createCmdPrefillButton(label, prefillText string, textEntryWidget *widget.Entry, textCanvas fyne.Canvas) *widget.Button {
	newButton := widget.NewButton(label, func() {
		prefillEntry(prefillText, textEntryWidget, textCanvas)
	})

	return newButton
}

func prefillEntry(prefillText string, textEntryWidget *widget.Entry, textCanvas fyne.Canvas) {
	textEntryWidget.SetText(prefillText)
	textEntryWidget.CursorColumn = len(prefillText)
	textEntryWidget.Refresh()
	textCanvas.Focus(textEntryWidget)
}

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
		above := fyne.NewPos(0, -popUp.Size().Height)
		popUp.ShowAtRelativePosition(above, newButton)
	})

	return newButton
}
