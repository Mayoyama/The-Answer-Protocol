package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"slices"
	"strings"
)

// resolveEmptyList writes the "None" line for an empty list reply, matched to the last
// INVENTORY, KEYITEMS or QUESTS command sent.
func (cr *CommandRouter) resolveEmptyList() {
	var emptyLine string

	switch cr.QueryWaitingForReply {
	case "KEYITEMS":
		emptyLine = "🔑 Key Items: None"
	case "INVENTORY":
		emptyLine = "🎒 Inventory: Empty"

		inventory := cr.Game.OverlayRightSection.InventoryList
		fyne.Do(func() {
			inventory.ItemIDs = nil
			inventory.List.Refresh()
		})

	case "QUESTS":
		emptyLine = "📜 Quests: None"
	default:
		return
	}

	cr.QueryWaitingForReply = ""

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(emptyLine)
	})
}

// handleChat writes a chat event to the chat window, labelling NPC senders and fight starts.
func (cr *CommandRouter) handleChat(reply ReplyContent) {
	var (
		sender      string
		chatString  string
		startBattle bool
	)

	if strings.HasPrefix(reply.Chat.Sender, "<<npc.") && strings.HasSuffix(reply.Chat.Sender, ">>") {
		startBattle = true

		nameOnly := strings.TrimSuffix(strings.TrimPrefix(reply.Chat.Sender, "<<"), ">>")
		sender = fmt.Sprintf("NPC (%s)", IDToLabel(nameOnly))

	} else {
		sender = fmt.Sprintf("(%s)", reply.Chat.Sender)
	}

	if startBattle && strings.Contains(reply.Chat.Message, "commenced a new battle") {
		chatString = fmt.Sprintf("⚔️ %s %s >> %s", Capitalize(reply.Chat.Scope), sender, reply.Chat.Message)

	} else {
		chatString = fmt.Sprintf("💬 %s %s >> %s", Capitalize(reply.Chat.Scope), sender, reply.Chat.Message)
	}

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.ChatWindow.addLine(chatString)
	})
}

// handleExamine writes an EXAMINE reply to the log.
func (cr *CommandRouter) handleExamine(reply ReplyContent) {
	examineString := fmt.Sprintf("🔍 Name: %s\nDescription: %s", reply.Examine.Name, reply.Examine.Description)

	if reply.Examine.Obtainable {
		examineString += "\nObtainable: Yes."
	}

	if reply.Examine.NPCRole != "" {
		examineString += fmt.Sprintf("\nNPC Role: %s.", SplitSnakeCase(reply.Examine.NPCRole))
	}

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(examineString)
	})
}

// handleGold updates the gold label.
func (cr *CommandRouter) handleGold(reply ReplyContent) {
	goldString := fmt.Sprintf("Gold: %d", reply.Gold)

	fyne.Do(func() {
		cr.Game.OverlayTopSection.GoldLabel.Text = goldString
		cr.Game.OverlayTopSection.GoldLabel.Refresh()
	})
}

// handleGroup logs group replies and events, and tracks whether the player is in a group
// (the In Party label and the party chat button).
func (cr *CommandRouter) handleGroup(reply ReplyContent) {
	var groupString string

	switch reply.Group.Action {
	case "CREATE":
		groupString = "👥 You created a new group!"
		cr.inGroup = true

	case "INVITE":
		groupString = fmt.Sprintf("📯 %s invites you to a group!", reply.Group.ByWho)

	case "JOIN":
		if cr.inGroup {
			groupString = fmt.Sprintf("🔗 %s joins the group.", reply.Group.ByWho)

		} else {
			groupString = fmt.Sprintf("🔗 You join %s's group.", reply.Group.ByWho)
			cr.inGroup = true
		}

	case "LEAVE":
		if reply.Group.ByWho == cr.PlayerUsername {
			groupString = "🍃 You leave the group."
			cr.inGroup = false

		} else {
			groupString = fmt.Sprintf("👋 %s leaves the group.", reply.Group.ByWho)
		}
	}

	inPT := cr.inGroup

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(groupString)

		if inPT {
			cr.Game.BottomSection.ChatPartyButton.Enable()
			cr.Game.OverlayTopSection.InPartyLabel.Text = "In Party: ✔️"
			cr.Game.OverlayTopSection.InPartyLabel.Refresh()

		} else {
			cr.Game.BottomSection.ChatPartyButton.Disable()
			cr.Game.OverlayTopSection.InPartyLabel.Text = "In Party: ✖️"
			cr.Game.OverlayTopSection.InPartyLabel.Refresh()
		}
	})
}

// handleInventory replaces the DROP list, and logs the inventory if the player asked for it.
func (cr *CommandRouter) handleInventory(reply ReplyContent) {
	var (
		items    []string
		itemList string
		waiting  bool
	)

	if strings.EqualFold(cr.QueryWaitingForReply, "inventory") {
		cr.QueryWaitingForReply = ""
		waiting = true
	}

	for _, item := range reply.Inventory {
		items = append(items, IDToLabel(item))
	}

	if len(items) > 0 {
		itemList = strings.Join(items, ", ")

	} else {
		itemList = "None"
	}

	itemsLine := fmt.Sprintf("🎒 Items: %s", itemList)
	inventory := cr.Game.OverlayRightSection.InventoryList
	inventoryIDs := reply.Inventory

	fyne.Do(func() {
		if waiting {
			cr.Game.OverlayBottomSection.LogWindow.addLine(itemsLine)
		}

		inventory.ItemIDs = inventoryIDs
		inventory.List.Refresh()
	})
}

// handleKeyItems logs the player's key items.
func (cr *CommandRouter) handleKeyItems(reply ReplyContent) {
	var keyItems []string

	if strings.EqualFold(cr.QueryWaitingForReply, "keyitems") {
		cr.QueryWaitingForReply = ""
	}

	for _, id := range reply.KeyItems {
		keyItems = append(keyItems, IDToLabel(id))
	}

	keyItemsList := "None"

	if len(keyItems) > 0 {
		keyItemsList = strings.Join(keyItems, ", ")
	}

	keyItemsLine := fmt.Sprintf("🔑 Key Items: %s", keyItemsList)

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(keyItemsLine)
	})
}

// handleLook updates the room name, the counters, the room details, the TAKE list, the NPC
// menus and the d-pad, and the zone image when the room changed.
func (cr *CommandRouter) handleLook(reply ReplyContent) {
	var (
		roomExits  []string
		npcLabels  []string
		itemLabels []string
	)

	for k, v := range reply.Look.Room.Exits {
		exit := fmt.Sprintf("%s: %s", Capitalize(k), IDToLabel(v))
		roomExits = append(roomExits, exit)
	}

	inRoomString := fmt.Sprintf("🏠 Room: %d ", len(reply.Look.Players))

	cr.InZoneCount = len(reply.Look.Players)

	for _, id := range reply.Look.NPCs {
		npcLabels = append(npcLabels, IDToLabel(id))
	}

	slices.Sort(reply.Look.Items)

	for _, item := range reply.Look.Items {
		itemLabels = append(itemLabels, IDToLabel(item))
	}

	slices.Sort(reply.Look.Players)
	slices.Sort(npcLabels)
	slices.Sort(roomExits)

	oTS := cr.Game.OverlayTopSection

	fyne.Do(func() {
		cr.Game.TopSection.ZoneNameLabel.Text = reply.Look.Room.Name
		cr.Game.TopSection.ZoneNameLabel.Refresh()

		cr.Game.TopSection.InRoomLabel.Text = inRoomString
		cr.Game.TopSection.InRoomLabel.Refresh()

		oTS.RoomStatsInfo.RoomDesc.SetText(reply.Look.Room.Description)
		oTS.RoomStatsInfo.PlayersList.SetText(strings.Join(reply.Look.Players, "\n"))
		oTS.RoomStatsInfo.NPCsList.SetText(strings.Join(npcLabels, "\n"))
		oTS.RoomStatsInfo.ExitsList.SetText(strings.Join(roomExits, "\n"))
		oTS.RoomStatsInfo.ItemsList.SetText(strings.Join(itemLabels, "\n"))

		changed := !slices.Equal(cr.Game.OverlayRightSection.TakeList.ItemIDs, reply.Look.Items)

		cr.Game.OverlayRightSection.TakeList.ItemIDs = reply.Look.Items
		cr.Game.OverlayRightSection.TakeList.List.Refresh()

		if changed {
			cr.Game.OverlayRightSection.TakeList.List.ScrollToBottom()
		}

		cr.Game.BottomSection.NPCTargetIDs = reply.Look.NPCs

		for direction, button := range cr.Game.OverlayRightSection.DPad.DirButtonSet {
			_, isExit := reply.Look.Room.Exits[direction]

			if isExit {
				button.Enable()

			} else {
				button.Disable()
			}
		}
	})

	if reply.Look.Room.ID != cr.CurrentZone {
		cr.CurrentZone = reply.Look.Room.ID
		cr.Game.GameScreen.showZoneImage(reply.Look.Room.ID)
	}
}

// handleMove changes the zone image after a move and requests LOOK.
func (cr *CommandRouter) handleMove(reply ReplyContent) {
	cr.CurrentZone = reply.RoomID

	cr.Game.GameScreen.showZoneImage(reply.RoomID)

	if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
		cr.Game.displayGameError(err.Error())
	}
}

// handleNPCDialogue writes NPC dialogue to the chat window, and requests STATUS after a
// healer's heal line.
func (cr *CommandRouter) handleNPCDialogue(reply ReplyContent) {
	var npcName string

	if reply.NPCName != "" {
		npcName = IDToLabel(reply.NPCName)

	} else if npcName == "" && cr.TalkTargetName != "" {
		npcName = IDToLabel(cr.TalkTargetName)

	} else {
		npcName = "NPC"
	}

	var npcLine string

	if reply.NPCHeal {
		npcLine = fmt.Sprintf("💖 NPC (%s): %s", npcName, reply.NPCText)

	} else {
		npcLine = fmt.Sprintf("🗣️ NPC (%s): %s", npcName, reply.NPCText)
	}

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.ChatWindow.addLine(npcLine)
	})

	if reply.NPCHeal {
		if err := cr.tc.sendButtonCommandToServer("STATUS"); err != nil {
			cr.Game.displayGameError(err.Error())
		}
	}
}

// handlePresence logs a player entering or leaving the room and adjusts the in-room counter.
func (cr *CommandRouter) handlePresence(reply ReplyContent) {
	var actionLine string

	if strings.EqualFold(reply.Presence.Direction, "enter") {
		actionLine = fmt.Sprintf("🚪⬅️ %s enters the room.", reply.Presence.Who)
		cr.InZoneCount++

	} else {
		actionLine = fmt.Sprintf("🚪➡️ %s leaves the room.", reply.Presence.Who)
		cr.InZoneCount--
	}

	inZoneText := fmt.Sprintf("🏠 Room: %d ", cr.InZoneCount)

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(actionLine)
		cr.Game.TopSection.InRoomLabel.Text = inZoneText
		cr.Game.TopSection.InRoomLabel.Refresh()
	})
}

// handleQuest logs an accepted quest with its rewards.
func (cr *CommandRouter) handleQuest(reply ReplyContent) {
	rewards := "Rewards: "

	if reply.Quest.Reward.Gold != 0 && reply.Quest.Reward.KeyItem != "" {
		rewards += fmt.Sprintf("Gold -> %d; Key Item -> %s", reply.Quest.Reward.Gold, IDToLabel(reply.Quest.Reward.KeyItem))

	} else if reply.Quest.Reward.Gold != 0 {
		rewards += fmt.Sprintf("Gold -> %d", reply.Quest.Reward.Gold)

	} else if reply.Quest.Reward.KeyItem != "" {
		rewards += fmt.Sprintf("Key Item -> %s", IDToLabel(reply.Quest.Reward.KeyItem))

	} else {
		rewards += "None"
	}

	questLine := fmt.Sprintf("📜 Quest Accepted!\nTask: %s\n%s", reply.Quest.Description, rewards)

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(questLine)
	})
}

// handleQuests logs the player's quest list.
func (cr *CommandRouter) handleQuests(reply ReplyContent) {
	var questList []string

	if strings.EqualFold(cr.QueryWaitingForReply, "quests") {
		cr.QueryWaitingForReply = ""
	}

	for _, q := range reply.Quests {
		questList = append(questList, buildQuestLine(q))
	}

	questsLine := "📜 Quests: \n" + strings.Join(questList, "\n")

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(questsLine)
	})
}

// handleReplyError shows a server error in the error label and the log.
func (cr *CommandRouter) handleReplyError(reply ReplyContent) {
	cr.Game.displayGameError(reply.Err.Text)
	cr.Quitting = false

	errLine := fmt.Sprintf("🔺 ERROR: %s (%d)", SplitSnakeCase(reply.Err.Text), reply.Err.Code)

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(errLine)
	})
}

// handleRespawn logs the defeat and respawn, and requests LOOK and STATUS.
func (cr *CommandRouter) handleRespawn(reply ReplyContent) {
	var enemyName string

	if cr.AttackTargetName == "" {
		enemyName = "Enemy NPC"

	} else {
		enemyName = IDToLabel(cr.AttackTargetName)
	}

	respawnLine := fmt.Sprintf("You were defeated by %s ⚰️. You wake up and mysteriously find yourself in %s.", enemyName, IDToLabel(reply.Respawned))

	if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
		cr.Game.displayGameError(err.Error())
	}

	if err := cr.tc.sendButtonCommandToServer("STATUS"); err != nil {
		cr.Game.displayGameError(err.Error())
	}

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(respawnLine)
	})
}

// handleRewards logs quest rewards, and requests GOLD after a gold reward to the player.
func (cr *CommandRouter) handleRewards(reply ReplyContent) {
	var (
		goldLine string
		kiLine   string
	)

	if reply.Rewards.Gold > 0 {
		goldLine = fmt.Sprintf("🎁 %s receives %d gold.", reply.RewardReceiver, reply.Rewards.Gold)
	}

	if reply.Rewards.KeyItem != "" {
		kiLine = fmt.Sprintf("🎁 %s receives %s.", reply.RewardReceiver, reply.Rewards.KeyItem)
	}

	fyne.Do(func() {
		if goldLine != "" {
			cr.Game.OverlayBottomSection.LogWindow.addLine(goldLine)
		}

		if kiLine != "" {
			cr.Game.OverlayBottomSection.LogWindow.addLine(kiLine)
		}
	})

	if goldLine != "" && cr.PlayerUsername == reply.RewardReceiver {
		if err := cr.tc.sendButtonCommandToServer("GOLD"); err != nil {
			cr.Game.displayGameError(err.Error())
		}
	}
}

// handleServerCount updates the online counter from EVT STATS.
func (cr *CommandRouter) handleServerCount(reply ReplyContent) {
	onlineCount := fmt.Sprintf("🌐 Online: %d ", reply.ServerCount)

	fyne.Do(func() {
		cr.Game.TopSection.OnlineLabel.Text = onlineCount
		cr.Game.TopSection.OnlineLabel.Refresh()
	})
}

// handleStatus updates the HP bar and the status label.
func (cr *CommandRouter) handleStatus(reply ReplyContent) {
	hp := reply.Status.HP
	maxHP := reply.Status.MaxHP
	status := fmt.Sprintf("Status: %s", reply.Status.Status)

	if hp > maxHP {
		hp = maxHP
	}

	fyne.Do(func() {
		cr.Game.TopSection.HPBar.Max = float64(maxHP)
		cr.Game.TopSection.HPBar.SetValue(float64(hp))
		cr.Game.OverlayTopSection.PlayerStatusLabel.Text = status
		cr.Game.OverlayTopSection.PlayerStatusLabel.Refresh()
	})
}

// handleTakeDrop logs a take or drop, updates the DROP list and requests LOOK.
func (cr *CommandRouter) handleTakeDrop(reply ReplyContent) {
	var actionLine string

	if strings.EqualFold(reply.TakeDrop.Action, "taken") {
		actionLine = "📥 You pick up a " + IDToLabel(reply.TakeDrop.Item)

	} else {
		actionLine = "📤 You drop the " + IDToLabel(reply.TakeDrop.Item)
	}

	inventory := cr.Game.OverlayRightSection.InventoryList

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(actionLine)

		if strings.EqualFold(reply.TakeDrop.Action, "taken") {
			inventory.ItemIDs = append(inventory.ItemIDs, reply.TakeDrop.Item)

		} else {
			var remaining []string

			for _, id := range inventory.ItemIDs {
				if id != reply.TakeDrop.Item {
					remaining = append(remaining, id)
				}
			}

			inventory.ItemIDs = remaining
		}

		inventory.List.Refresh()
	})

	if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
		cr.Game.displayGameError(err.Error())
	}
}

// handleWho logs the WHO reply and updates the online counter.
func (cr *CommandRouter) handleWho(reply ReplyContent) {
	whoString := fmt.Sprintf("👤 In zone: %v. Online Count: %d.", reply.Who.Room, reply.Who.Server)
	onlineCount := fmt.Sprintf("🌐 Online: %d ", reply.Who.Server)

	fyne.Do(func() {
		cr.Game.TopSection.OnlineLabel.Text = onlineCount
		cr.Game.TopSection.OnlineLabel.Refresh()
		cr.Game.OverlayBottomSection.LogWindow.addLine(whoString)
	})
}
