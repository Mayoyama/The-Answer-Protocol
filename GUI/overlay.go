package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strings"
)

type OverlayTopSection struct {
	RoomDetailsButton *widget.Button
	BattleLogButton   *widget.Button
	GoldLabel         *canvas.Text
	PlayerStatusLabel *canvas.Text
	StatsGoldContent  *fyne.Container

	RoomStatsInfo   *RoomStatsWidget
	ZoneLookContent *fyne.Container

	DPad *DPad

	BattleLog        *ChatWindow
	BattleLogContent *fyne.Container
	BattleSwapStack  *fyne.Container
}

type OverlayRightSection struct {
	DPad          *DPad
	InventoryList *ItemList
	TakeList      *ItemList
}

type OverlayBottomSection struct {
	ChatWindow *ChatWindow
	LogWindow  *ChatWindow
	TextEntry  *widget.Entry
}

const WindowRectAlpha uint8 = 135

func buildTopOverlay(tapConnRequests chan<- string) *OverlayTopSection {
	const (
		battleLogWidth  float32 = 760
		battleLogHeight float32 = 250
	)

	oTS := &OverlayTopSection{}

	oTS.PlayerStatusLabel = canvas.NewText("Status: -", color.White)
	oTS.PlayerStatusLabel.Alignment = fyne.TextAlignLeading
	oTS.GoldLabel = canvas.NewText("Gold: -", color.White)
	oTS.GoldLabel.Alignment = fyne.TextAlignLeading
	statsGoldBox := container.NewVBox(oTS.PlayerStatusLabel, oTS.GoldLabel)
	statsGoldWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})
	oTS.StatsGoldContent = container.NewStack(statsGoldWinBacking, statsGoldBox)

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
	infoBg := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})
	infoBg.CornerRadius = 6
	infoContent := container.NewVBox(oTS.RoomStatsInfo.RoomDesc, roomStatsGrid)
	oTS.ZoneLookContent = container.NewStack(infoBg, infoContent)

	oTS.BattleLog = buildChatWindow(battleLogWidth, battleLogHeight)

	battleBg := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})
	battleBg.CornerRadius = 6

	oTS.BattleLogContent = container.NewStack(battleBg, oTS.BattleLog.Scroll)
	oTS.BattleLogContent.Hide()

	oTS.BattleSwapStack = container.NewStack(oTS.ZoneLookContent, oTS.BattleLogContent)

	oTS.RoomDetailsButton = widget.NewButton("LOOK", func() {
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

	oTS.BattleLogButton = widget.NewButton("Battle Log", func() {
		if oTS.BattleLogContent.Visible() {
			oTS.BattleLogContent.Hide()

		} else {
			oTS.ZoneLookContent.Hide()
			oTS.BattleLogContent.Show()
		}

		oTS.BattleSwapStack.Refresh()
		oTS.BattleLog.Scroll.ScrollToBottom()
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

	dPadRow := container.NewHBox(layout.NewSpacer(), dPad)

	InvTitle := widget.NewLabel("DROP")
	InvTitle.Alignment = fyne.TextAlignCenter
	oRS.InventoryList = buildItemList("DROP", tapConnRequests)
	invRowHeight := widget.NewLabel("").MinSize().Height
	invContent := container.NewGridWrap(fyne.NewSize(190, invRowHeight*4), oRS.InventoryList.List)
	invBox := container.NewBorder(InvTitle, nil, nil, nil, invContent)

	takeItemsTitle := widget.NewLabel("TAKE")
	takeItemsTitle.Alignment = fyne.TextAlignCenter
	oRS.TakeList = buildItemList("TAKE", tapConnRequests)
	takeItemsRowHeight := widget.NewLabel("").MinSize().Height
	takeItemsContent := container.NewGridWrap(fyne.NewSize(190, takeItemsRowHeight*4), oRS.TakeList.List)
	takeItemsBox := container.NewBorder(takeItemsTitle, nil, nil, nil, takeItemsContent)

	invWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})
	takeWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})

	invWinStack := container.NewStack(invWinBacking, invBox)
	takeWinStack := container.NewStack(takeWinBacking, takeItemsBox)

	rightSideContent := container.NewHBox(layout.NewSpacer(), container.NewVBox(dPadRow, layout.NewSpacer(), invWinStack, takeWinStack))

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
					toChatWin := fmt.Sprintf("💬 %s (YOU) << %s", scope, cmdParts[2])
					oBS.ChatWindow.addLine(toChatWin)
				}

			}
		default:
			// TODO: show "buffer full" on GameErr (the section does not have gs; decide how to reach it)
		}

		oBS.TextEntry.SetText((""))
	}

	chatWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})
	logWinBacking := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: WindowRectAlpha})

	chatStack := container.NewStack(chatWinBacking, oBS.ChatWindow.Scroll)
	logStack := container.NewStack(logWinBacking, oBS.LogWindow.Scroll)

	chatWindows := container.NewHBox(chatStack, logStack)

	content := container.New(layout.NewCustomPaddedVBoxLayout(0), chatWindows, oBS.TextEntry)

	return oBS, container.NewHBox(content)
}
