package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"slices"
	"strings"
	"time"
)

func (cr *CommandRouter) handleChat(reply ReplyContent) {
	chatString := fmt.Sprintf("%s (%s) >> %s", Capitalize(reply.Chat.Scope), reply.Chat.Sender, reply.Chat.Message)

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.ChatWindow.addLine(chatString)
	})
}

func (cr *CommandRouter) handleExamine(reply ReplyContent) {
	examineString := fmt.Sprintf("Name: %s. Description: %s.", reply.Examine.Name, reply.Examine.Description)

	if reply.Examine.Obtainable {
		examineString += " Obtainable: Yes."
	}

	if reply.Examine.NPCRole != "" {
		examineString += fmt.Sprintf(" NPC Role: %s.", SplitSnakeCase(reply.Examine.NPCRole))
	}

	fyne.Do(func() {
		cr.Game.OverlayBottomSection.LogWindow.addLine(examineString)
	})
}

func (cr *CommandRouter) handleGold(reply ReplyContent) {
	goldString := fmt.Sprintf("Gold: %d", reply.Gold)

	fyne.Do(func() {
		cr.Game.OverlayTopSection.GoldLabel.SetText(goldString)
	})
}

func (cr *CommandRouter) handleLook(reply ReplyContent) {
	var (
		roomExits  []string
		npcLabels  []string
		itemLabels []string
	)

	for k, v := range reply.Look.Room.Exits {
		exit := fmt.Sprintf("%s: %s", Capitalize(k), SplitSnakeCase(v))
		roomExits = append(roomExits, exit)
	}

	slices.Sort(roomExits)

	inRoomString := fmt.Sprintf("Room: %d", len(reply.Look.Players))

	cr.InZoneCount = len(reply.Look.Players)

	for _, id := range reply.Look.NPCs {
		npcLabels = append(npcLabels, IDToLabel(id))
	}

	for _, id := range reply.Look.Items {
		itemLabels = append(itemLabels, IDToLabel(id))
	}

	fyne.Do(func() {
		cr.Game.TopSection.ZoneNameLabel.Text = reply.Look.Room.Name
		cr.Game.TopSection.ZoneNameLabel.Refresh()

		cr.Game.TopSection.InRoomLabel.Text = inRoomString
		cr.Game.TopSection.InRoomLabel.Refresh()

		cr.Game.OverlayTopSection.RoomStatsInfo.RoomDesc.SetText(reply.Look.Room.Description)
		cr.Game.OverlayTopSection.RoomStatsInfo.PlayersList.SetText(strings.Join(reply.Look.Players, "\n"))
		cr.Game.OverlayTopSection.RoomStatsInfo.NPCsList.SetText(strings.Join(npcLabels, "\n"))
		cr.Game.OverlayTopSection.RoomStatsInfo.ExitsList.SetText(strings.Join(roomExits, "\n"))
		cr.Game.OverlayTopSection.RoomStatsInfo.ItemsList.SetText(strings.Join(itemLabels, "\n"))
	})

	if reply.Look.Room.ID != cr.CurrentZone {
		cr.CurrentZone = reply.Look.Room.ID
		cr.Game.GameScreen.showZoneImage(reply.Look.Room.ID)
	}
}

func (cr *CommandRouter) handleMove(reply ReplyContent) {
	cr.CurrentZone = reply.RoomID

	cr.Game.GameScreen.showZoneImage(reply.RoomID)

	if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
		cr.Game.displayGameError(err.Error())
	}
}

func (cr *CommandRouter) handleRespawn() {
	if err := cr.tc.sendButtonCommandToServer("LOOK"); err != nil {
		cr.Game.displayGameError(err.Error())
	}

	if err := cr.tc.sendButtonCommandToServer("STATUS"); err != nil {
		cr.Game.displayGameError(err.Error())
	}
}

func (cr *CommandRouter) handleReplyError(reply ReplyContent) {
	cr.Game.displayGameError(reply.Err.Text)

	if reply.Err.Code == 750 {
		lockDuration := time.Minute

		_, timeString, found := strings.Cut(reply.Err.Text, "Time remaining: ")

		if found {
			penaltyTime, parseErr := time.ParseDuration(strings.TrimSpace(timeString))

			if parseErr == nil {
				lockDuration = penaltyTime
			}
		}

		if cr.Game.LockoutTimer != nil {
			cr.Game.LockoutTimer.Stop()
		}

		fyne.Do(func() {
			for _, button := range cr.Game.OverlayRightSection.DPad.DirButtonSet {
				button.Disable()
			}
		})

		cr.Game.LockoutTimer = time.AfterFunc(lockDuration, func() {
			fyne.Do(func() {
				for _, button := range cr.Game.OverlayRightSection.DPad.DirButtonSet {
					button.Enable()
				}
			})
		})
	}
}

func (cr *CommandRouter) handleServerCount(reply ReplyContent) {
	onlineCount := fmt.Sprintf("Online: %d ", reply.ServerCount)

	fyne.Do(func() {
		cr.Game.TopSection.OnlineLabel.Text = onlineCount
		cr.Game.TopSection.OnlineLabel.Refresh()
	})
}

func (cr *CommandRouter) handleStatus(reply ReplyContent) {
	if reply.HPHeal.HealMessage != "" {
		// TODO: write reply.HPHeal.HealMessage to the log once the log window exists

		if err := cr.tc.sendButtonCommandToServer("STATUS"); err != nil {
			cr.Game.displayGameError(err.Error())
		}

	} else {
		hp := reply.Status.HP
		maxHP := reply.Status.MaxHP
		status := fmt.Sprintf("Status: %s", reply.Status.Status)

		if hp > maxHP {
			hp = maxHP
		}

		fyne.Do(func() {
			cr.Game.TopSection.HPBar.Max = float64(maxHP)
			cr.Game.TopSection.HPBar.SetValue(float64(hp))
			cr.Game.OverlayTopSection.PlayerStatusLabel.SetText(status)
		})
	}
}

func (cr *CommandRouter) handleWho(reply ReplyContent) {
	//whoString := fmt.Sprintf("In zone: %v. Online Count: %d.", reply.Who.Room, reply.Who.Server)
	onlineCount := fmt.Sprintf("Online: %d ", reply.Who.Server)

	fyne.Do(func() {
		cr.Game.TopSection.OnlineLabel.Text = onlineCount
		cr.Game.TopSection.OnlineLabel.Refresh()
		// TODO: write whoString to the log once the log window exists
	})
}
