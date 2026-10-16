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

	BottomSection *GSBottomSection
	BottomContent *fyne.Container

	ErrTimer     *time.Timer
	LockoutTimer *time.Timer

	ShowHideButton *widget.Button

	GameErr *widget.Label
}

type GSTopSection struct {
	HPBar         *widget.ProgressBar
	ZoneNameLabel *canvas.Text
	OnlineLabel   *canvas.Text
	InRoomLabel   *canvas.Text
}

type GSBottomSection struct {
	ChatGlobalButton *widget.Button
	ChatRoomButton   *widget.Button
	ChatPartyButton  *widget.Button
	ClearChatButton  *widget.Button
	ChatContent      *fyne.Container

	PartyCreateButton *widget.Button
	PartyInviteButton *widget.Button
	PartyJoinButton   *widget.Button
	PartyLeaveButton  *widget.Button
	PartyContent      *fyne.Container

	TalkButton     *widget.Button
	QuestButton    *widget.Button
	AttackButton   *widget.Button
	FleeButton     *widget.Button
	MiddleContent  *fyne.Container
	TargetsContent *fyne.Container

	KeyItemButton       *widget.Button
	InventoryButton     *widget.Button
	QuestsButton        *widget.Button
	WhoButton           *widget.Button
	QueryButtonsContent *fyne.Container

	QuitButton   *widget.Button
	RightContent *fyne.Container

	NPCTargetIDs []string
	SelectedCmd  string
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

func (gs *GameScreen) buildBottomSection(tapConnRequests chan<- string, winCanvas fyne.Canvas) {
	gs.BottomSection = &GSBottomSection{}

	gsBS := gs.BottomSection
	oBS := gs.OverlayBottomSection

	gsBS.ChatGlobalButton = createCmdPrefillButton("G", "CHAT GLOBAL ", oBS.TextEntry, winCanvas)
	gsBS.ChatRoomButton = createCmdPrefillButton("R", "CHAT ROOM ", oBS.TextEntry, winCanvas)
	gsBS.ChatPartyButton = createCmdPrefillButton("P", "CHAT GROUP ", oBS.TextEntry, winCanvas)
	gsBS.ClearChatButton = createCmdPrefillButton("C", "", oBS.TextEntry, winCanvas)
	gsBS.ChatContent = container.NewHBox(gsBS.ChatGlobalButton, gsBS.ChatRoomButton, gsBS.ChatPartyButton, gsBS.ClearChatButton)

	partyLabel := widget.NewLabel("GROUP")
	gsBS.PartyCreateButton = createStatusButton("CREATE", "GROUP CREATE", tapConnRequests)
	gsBS.PartyInviteButton = createCmdPrefillButton("INVITE", "GROUP INVITE ", oBS.TextEntry, winCanvas)
	gsBS.PartyJoinButton = createCmdPrefillButton("JOIN", "GROUP JOIN ", oBS.TextEntry, winCanvas)
	gsBS.PartyLeaveButton = createStatusButton("LEAVE", "GROUP LEAVE", tapConnRequests)
	partyButtons := container.NewGridWithColumns(2, gsBS.PartyCreateButton, gsBS.PartyJoinButton, gsBS.PartyInviteButton, gsBS.PartyLeaveButton)

	gsBS.PartyContent = container.NewVBox(layout.NewSpacer(), partyLabel, partyButtons, layout.NewSpacer())

	leftContent := container.NewVBox(gs.BottomSection.ChatContent, gs.BottomSection.PartyContent)

	gsBS.TalkButton = createPopUpMenuButton("TALK", "TALK", &gsBS.NPCTargetIDs, tapConnRequests, winCanvas)
	gsBS.QuestButton = createPopUpMenuButton("QUEST", "QUEST", &gsBS.NPCTargetIDs, tapConnRequests, winCanvas)
	gsBS.AttackButton = createPopUpMenuButton("ATTACK", "ATTACK", &gsBS.NPCTargetIDs, tapConnRequests, winCanvas)
	gsBS.FleeButton = createStatusButton("FLEE", "FLEE", tapConnRequests)
	actionButtons := container.NewGridWithColumns(2, gsBS.TalkButton, gsBS.AttackButton, gsBS.QuestButton, gsBS.FleeButton)

	interactionsLabel := widget.NewLabel("INTERACTIONS")
	gsBS.MiddleContent = container.NewVBox(layout.NewSpacer(), interactionsLabel, actionButtons)

	gsBS.KeyItemButton = createStatusButton("Key Items", "KEYITEMS", tapConnRequests)
	gsBS.InventoryButton = createStatusButton("Inventory", "INVENTORY", tapConnRequests)
	gsBS.QuestsButton = createStatusButton("Quests", "QUESTS", tapConnRequests)
	gsBS.WhoButton = createStatusButton("Who", "WHO", tapConnRequests)

	queriesLabel := widget.NewLabel("QUERIES")
	queryButtons := container.NewGridWithColumns(2, gsBS.InventoryButton, gsBS.QuestsButton, gsBS.KeyItemButton, gsBS.WhoButton)
	gsBS.QueryButtonsContent = container.NewVBox(layout.NewSpacer(), queriesLabel, queryButtons)

	mainSections := container.NewGridWithColumns(3, leftContent, gsBS.MiddleContent, gsBS.QueryButtonsContent)

	gsBS.QuitButton = createStatusButton("👋", "QUIT", tapConnRequests)

	rightButtonsBox := container.NewVBox(gs.ShowHideButton, layout.NewSpacer(), gsBS.QuitButton)

	gs.BottomContent = container.NewBorder(nil, nil, nil, rightButtonsBox, mainSections)

	gsBS.FleeButton.Disable()
}

func buildGameScreen(tapConnRequests chan<- string, winCanvas fyne.Canvas) *GameScreen {
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

	overlayLayers := []fyne.CanvasObject{overlayContent, statsGoldLayer, gs.OverlayRightContent}

	gs.ShowHideButton = createShowHideButton(overlayLayers)

	gs.buildBottomSection(tapConnRequests, winCanvas)

	gs.GameContent = container.NewVBox(gs.TopRowContent, overlayStack, gs.BottomContent)

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
