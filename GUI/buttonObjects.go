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

func createStatusButton(label, cmd string, cmdChan chan<- string) *widget.Button {
	newButton := widget.NewButton(label, func() {
		select {
		case cmdChan <- cmd:
		default:
		}
	})

	return newButton
}

func createCmdPrefillButton(label, prefillText string, textEntryWidget *widget.Entry, textCanvas fyne.Canvas) *widget.Button {
	newButton := widget.NewButton(label, func() {
		textEntryWidget.SetText(prefillText)
		textEntryWidget.CursorColumn = len(prefillText)
		textEntryWidget.Refresh()
		textCanvas.Focus(textEntryWidget)
	})

	return newButton
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

func createPopUpMenuButton(label, cmd string, menuListContent *[]string, cmdChan chan<- string, winCanvas fyne.Canvas) *widget.Button {
	var newButton *widget.Button

	newButton = widget.NewButton(label, func() {
		var menuItems []*fyne.MenuItem

		for _, id := range *menuListContent {
			item := fyne.NewMenuItem(IDToLabel(id), func() {
				select {
				case cmdChan <- cmd + " " + id:
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
