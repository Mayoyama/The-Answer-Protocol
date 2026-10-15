package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func buildItemList(cmd string, tapConnRequests chan<- string) *ItemList {
	iList := &ItemList{}

	iList.List = widget.NewList(
		func() int {
			return len(iList.ItemIDs)
		},
		func() fyne.CanvasObject {
			rowLabel := widget.NewLabel("")
			rowLabel.Alignment = fyne.TextAlignCenter

			return rowLabel
		},
		func(id widget.ListItemID, row fyne.CanvasObject) {
			row.(*widget.Label).SetText(IDToLabel(iList.ItemIDs[id]))
		},
	)

	iList.List.OnSelected = func(id widget.ListItemID) {
		itemID := iList.ItemIDs[id]

		select {
		case tapConnRequests <- cmd + " " + itemID:
		default:
		}

		iList.List.UnselectAll()
	}

	return iList
}
