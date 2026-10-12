package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type OverlayTopSection struct {
	RoomDetailsButton *widget.Button
	GoldLabel         *widget.Label
	PlayerStatusLabel *widget.Label
	StatsGoldContent  *fyne.Container

	RoomStatsInfo   *RoomStatsWidget
	ZoneLookContent *fyne.Container

	DPad *DPad
}

type OverlayRightSection struct {
	DPad *DPad
}

type OverlayBottomSection struct {
	ChatWindow *ChatWindow
	LogWindow  *ChatWindow
	TextEntry  *widget.Entry
}

func buildTopOverlay() *OverlayTopSection {
	oTS := &OverlayTopSection{}

	oTS.PlayerStatusLabel = widget.NewLabel("Status: -")
	oTS.PlayerStatusLabel.Alignment = fyne.TextAlignLeading
	oTS.GoldLabel = widget.NewLabel("Gold: -")
	oTS.GoldLabel.Alignment = fyne.TextAlignLeading
	oTS.StatsGoldContent = container.NewVBox(oTS.PlayerStatusLabel, oTS.GoldLabel)

	oTS.RoomStatsInfo = &RoomStatsWidget{}

	playerTitle := widget.NewLabel("PLAYERS")
	playerTitle.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.PlayersList = widget.NewLabel("")
	oTS.RoomStatsInfo.PlayersList.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.PlayersList.Wrapping = fyne.TextWrapBreak
	playersRowHeight := oTS.RoomStatsInfo.PlayersList.MinSize().Height
	playersScroll := container.NewVScroll(oTS.RoomStatsInfo.PlayersList)
	playersScroll.SetMinSize(fyne.NewSize(190, playersRowHeight*4))
	playersColumn := container.NewBorder(playerTitle, nil, nil, nil, playersScroll)

	npcsTitle := widget.NewLabel("NPCS")
	npcsTitle.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.NPCsList = widget.NewLabel("")
	oTS.RoomStatsInfo.NPCsList.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.NPCsList.Wrapping = fyne.TextWrapBreak
	npcsRowHeight := oTS.RoomStatsInfo.NPCsList.MinSize().Height
	npcsScroll := container.NewVScroll(oTS.RoomStatsInfo.NPCsList)
	npcsScroll.SetMinSize(fyne.NewSize(190, npcsRowHeight*4))
	npcsColumn := container.NewBorder(npcsTitle, nil, nil, nil, npcsScroll)

	exitsTitle := widget.NewLabel("EXITS")
	exitsTitle.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.ExitsList = widget.NewLabel("")
	oTS.RoomStatsInfo.ExitsList.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.ExitsList.Wrapping = fyne.TextWrapBreak
	exitsRowHeight := oTS.RoomStatsInfo.ExitsList.MinSize().Height
	exitsScroll := container.NewVScroll(oTS.RoomStatsInfo.ExitsList)
	exitsScroll.SetMinSize(fyne.NewSize(190, exitsRowHeight*4))
	exitsColumn := container.NewBorder(exitsTitle, nil, nil, nil, exitsScroll)

	itemsTitle := widget.NewLabel("ITEMS")
	itemsTitle.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.ItemsList = widget.NewLabel("")
	oTS.RoomStatsInfo.ItemsList.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.ItemsList.Wrapping = fyne.TextWrapBreak
	itemsRowHeight := oTS.RoomStatsInfo.ItemsList.MinSize().Height
	itemsScroll := container.NewVScroll(oTS.RoomStatsInfo.ItemsList)
	itemsScroll.SetMinSize(fyne.NewSize(190, itemsRowHeight*4))
	itemsColumn := container.NewBorder(itemsTitle, nil, nil, nil, itemsScroll)

	oTS.RoomStatsInfo.RoomDesc = widget.NewLabel("")
	oTS.RoomStatsInfo.RoomDesc.Alignment = fyne.TextAlignCenter
	oTS.RoomStatsInfo.RoomDesc.Wrapping = fyne.TextWrapWord

	roomStatsGrid := container.NewGridWithColumns(4, playersColumn, npcsColumn, exitsColumn, itemsColumn)
	infoBg := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 120})
	infoBg.CornerRadius = 6
	infoContent := container.NewVBox(oTS.RoomStatsInfo.RoomDesc, roomStatsGrid)
	oTS.ZoneLookContent = container.NewStack(infoBg, infoContent)

	oTS.RoomDetailsButton = widget.NewButton("Room Details", func() {
		if oTS.ZoneLookContent.Visible() {
			oTS.ZoneLookContent.Hide()

		} else {
			oTS.ZoneLookContent.Show()
		}

		oTS.ZoneLookContent.Refresh()
	})

	return oTS
}

func buildRightOverlay(tapConnRequests chan<- string) (*OverlayRightSection, *fyne.Container) {
	oRS := &OverlayRightSection{}
	oRS.DPad = &DPad{}

	oRS.DPad.DirButtonSet = createDirButtonSet(tapConnRequests)

	dPad := container.NewGridWithColumns(
		3,
		widget.NewLabel(""), oRS.DPad.DirButtonSet["north"], oRS.DPad.DirButtonSet["up"],
		oRS.DPad.DirButtonSet["west"], widget.NewLabel(""), oRS.DPad.DirButtonSet["east"],
		widget.NewLabel(""), oRS.DPad.DirButtonSet["south"], oRS.DPad.DirButtonSet["down"],
	)

	rightSideContent := container.NewHBox(layout.NewSpacer(), container.NewVBox(dPad, layout.NewSpacer()))

	return oRS, rightSideContent
}

func buildBottomOverlay(tapConnRequests chan<- string) (*OverlayBottomSection, *fyne.Container) {
	oBS := &OverlayBottomSection{}
	oBS.ChatWindow = buildChatWindow(350, 250)
	oBS.LogWindow = buildChatWindow(350, 250)

	oBS.TextEntry = widget.NewEntry()
	oBS.TextEntry.SetPlaceHolder("TAP >>")

	oBS.TextEntry.OnSubmitted = func(input string) {
		cmd := strings.TrimSpace(input)

		if cmd == "" {
			return
		}

		cmdParts := strings.SplitN(cmd, " ", 3)
		cmdUpper := strings.ToUpper(cmdParts[0])

		select {
		case tapConnRequests <- cmd:
			if len(cmdParts) > 2 && cmdUpper == "CHAT" {
				scope := Capitalize(cmdParts[1])
				if scope == "Room" || scope == "Global" || scope == "Group" {
					toChatWin := fmt.Sprintf("%s (YOU) << %s", scope, cmdParts[2])
					oBS.ChatWindow.addLine(toChatWin)
				}

			}
		default:
			// TODO: show "buffer full" on GameErr (the section does not have gs; decide how to reach it)
		}

		oBS.TextEntry.SetText((""))
	}

	chatWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 125})
	logWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 125})

	chatStack := container.NewStack(chatWinBacking, oBS.ChatWindow.Scroll)
	logStack := container.NewStack(logWinBacking, oBS.LogWindow.Scroll)

	chatWindows := container.NewHBox(chatStack, logStack)

	textBox := container.NewGridWithColumns(2, oBS.TextEntry, layout.NewSpacer())

	content := container.NewVBox(chatWindows, textBox)

	return oBS, content
}
