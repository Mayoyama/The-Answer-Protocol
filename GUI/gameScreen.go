package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"time"
)

type GameScreen struct {
	GameContent       *fyne.Container
	GameScreen        *Screen
	ZoneNameDescLabel *widget.Label
	GameErr           *widget.Label
	DirButtonSet      map[string]*widget.Button

	ErrTimer     *time.Timer
	LockoutTimer *time.Timer
}

func buildGameScreen(tapConnRequests chan<- string) *GameScreen {
	gs := &GameScreen{}

	gs.GameErr = widget.NewLabel("")
	gs.GameErr.Wrapping = fyne.TextWrapWord

	gs.DirButtonSet = createDirButtonSet(tapConnRequests)

	dPad := container.NewGridWithColumns(
		3, widget.NewLabel(""), gs.DirButtonSet["north"], gs.DirButtonSet["up"],
		gs.DirButtonSet["west"], widget.NewLabel(""), gs.DirButtonSet["east"],
		widget.NewLabel(""), gs.DirButtonSet["south"], gs.DirButtonSet["down"])

	gs.GameScreen = buildNewZoneImage(backdrop_w, backdrop_h)

	gs.ZoneNameDescLabel = widget.NewLabel("")
	gs.ZoneNameDescLabel.Alignment = fyne.TextAlignCenter
	gs.ZoneNameDescLabel.Wrapping = fyne.TextWrapWord

	gs.GameContent = container.NewVBox(gs.GameScreen.zoneStack, gs.ZoneNameDescLabel, dPad, gs.GameErr)

	return gs
}

func (gs *GameScreen) displayGameError(errMsg string) {
	if gs.ErrTimer != nil {
		gs.ErrTimer.Stop()
	}

	fyne.Do(func() {
		gs.GameErr.SetText(errMsg)
	})

	gs.ErrTimer = time.AfterFunc(time.Second*3, func() {
		fyne.Do(func() {
			gs.GameErr.SetText("")
		})
	})
}
