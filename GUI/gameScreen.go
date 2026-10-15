package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"time"
)

type GameScreen struct {
	GameScreen  *Screen
	GameContent *fyne.Container

	TopSection    *GSTopSection
	TopRowContent *fyne.Container

	OverlayTopSection *OverlayTopSection

	OverlayRightSection *OverlayRightSection
	OverlayRightContent *fyne.Container

	OverlayBottomSection *OverlayBottomSection
	OverlayBottomContent *fyne.Container

	ErrTimer     *time.Timer
	LockoutTimer *time.Timer

	GameErr *widget.Label
}

type GSTopSection struct {
	HPBar         *widget.ProgressBar
	ZoneNameLabel *canvas.Text
	OnlineLabel   *canvas.Text
	InRoomLabel   *canvas.Text
}

type RoomStatsWidget struct {
	PlayersList *widget.Label
	NPCsList    *widget.Label
	ExitsList   *widget.Label
	ItemsList   *widget.Label
	RoomDesc    *widget.Label
}

type DPad struct {
	DirButtonSet map[string]*widget.Button
}

func buildTopSection() (*GSTopSection, *fyne.Container) {
	gsTop := &GSTopSection{}

	hpBarLabel := canvas.NewText("HP", color.White)
	gsTop.HPBar = widget.NewProgressBar()
	gsTop.HPBar.TextFormatter = func() string {
		return fmt.Sprintf("%.0f/%.0f", gsTop.HPBar.Value, gsTop.HPBar.Max)
	}

	hpBarWrap := container.NewGridWrap(fyne.NewSize(160, 20), gsTop.HPBar)
	hpBarRow := container.NewHBox(container.NewCenter(hpBarLabel), hpBarWrap)
	hpBarCell := container.NewVBox(layout.NewSpacer(), hpBarRow, layout.NewSpacer())

	gsTop.ZoneNameLabel = canvas.NewText("-", color.White)
	gsTop.ZoneNameLabel.TextSize = 20
	gsTop.ZoneNameLabel.TextStyle = fyne.TextStyle{Bold: true}
	gsTop.ZoneNameLabel.Alignment = fyne.TextAlignCenter

	gsTop.OnlineLabel = canvas.NewText("🌐 Online: - ", color.White)
	gsTop.OnlineLabel.Alignment = fyne.TextAlignTrailing
	gsTop.InRoomLabel = canvas.NewText("🏠 Room: - ", color.White)
	gsTop.InRoomLabel.Alignment = fyne.TextAlignTrailing
	onlineContent := container.NewVBox(layout.NewSpacer(), gsTop.OnlineLabel, gsTop.InRoomLabel, layout.NewSpacer())

	topRowContent := container.NewGridWithColumns(3, hpBarCell, gsTop.ZoneNameLabel, onlineContent)

	return gsTop, topRowContent
}

func buildGameScreen(tapConnRequests chan<- string) *GameScreen {
	gs := &GameScreen{}

	gs.TopSection, gs.TopRowContent = buildTopSection()

	gs.GameErr = widget.NewLabel("")
	gs.GameErr.Wrapping = fyne.TextWrapWord

	gameErrorContent := container.NewVBox(layout.NewSpacer(), gs.GameErr, layout.NewSpacer())

	gs.GameScreen = buildNewZoneImage(backdrop_w, backdrop_h)

	gs.OverlayTopSection = buildTopOverlay(tapConnRequests) //, gs.OverlayTopContent

	gs.OverlayRightSection, gs.OverlayRightContent = buildRightOverlay(tapConnRequests)

	gs.OverlayBottomSection, gs.OverlayBottomContent = buildBottomOverlay(tapConnRequests)

	roomInfoCentered := container.NewCenter(container.NewHBox(
		container.NewGridWrap(fyne.NewSize(140, 25), gs.OverlayTopSection.RoomDetailsButton),
		container.NewGridWrap(fyne.NewSize(140, 25), gs.OverlayTopSection.BattleLogButton)))

	roomInfoButtonBox := container.NewVBox(roomInfoCentered, container.NewCenter(gs.OverlayTopSection.BattleSwapStack))

	overlayContent := container.NewBorder(roomInfoButtonBox, gs.OverlayBottomContent, gameErrorContent, nil)
	
	statsGoldLayer := container.NewVBox(container.NewHBox(gs.OverlayTopSection.StatsGoldContent))

	overlayStack := container.NewStack(gs.GameScreen.zoneStack, overlayContent, statsGoldLayer, gs.OverlayRightContent)

	gs.GameContent = container.NewVBox(gs.TopRowContent, overlayStack)

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
