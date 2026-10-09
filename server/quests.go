package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// QuestType categorizes a quest's objective (delivery, fetch, or battle).
type QuestType int

// QuestType values
const (
	Delivery QuestType = iota
	Fetch
	Battle
)

// QuestStatus is a player's progress state for one accepted quest.
type QuestStatus int

// QuestStatus values
const (
	Active QuestStatus = iota
	Completed
)

// Quest represents an in-game quest.
type Quest struct {
	Name        string
	QuestID     string
	Description string
	Type        QuestType
	Steps       []QuestAction
	Reward      *Reward
	Requires    string // bare quest key of a prerequisite quest; "" if none
}

// Reward is a quest's payout: an optional key item and/or gold.
// Used both when parsing world.yaml and when serializing a QUEST response.
type Reward struct {
	KeyItem string `yaml:"key_item" json:"key_item,omitempty"`
	Money   int    `yaml:"gold" json:"gold,omitempty"`
}

// PlayerQuest tracks one player's progress on a single accepted quest.
type PlayerQuest struct {
	Quest     *Quest
	StepIndex int
	Status    QuestStatus
}

// printPlayerQuests sends the player's accepted quests (with progress and quest_items)
// as a JSON response.
func printPlayerQuests(player *Player) {
	pname := player.getPlayerName()

	player.PlayerMu.Lock()

	questResponse := make(QuestsResponse, 0, len(player.Quests))

	for _, q := range player.Quests {
		var statusString string

		switch q.Status {
		case Active:
			statusString = "active"

		case Completed:
			statusString = "completed"

		default:
			statusString = "unknown"
		}

		var (
			progress       string
			newSumm        QuestSummary
			questItemCount string
		)

		if q.Status == Active {
			progress = fmt.Sprintf("%d/%d", q.StepIndex, len(q.Quest.Steps))

			step, isTTA := q.Quest.Steps[q.StepIndex].(*TalkToAction)

			if isTTA && len(step.ReceivesKeyItems) > 0 {
				alreadyHeld := 0

				for _, ki := range step.ReceivesKeyItems {
					keyItem, ok := resolveKeyItem(ki)

					if ok && player.KeyInventory[keyItem.ItemID] {
						alreadyHeld++
					}
				}

				questItemCount = fmt.Sprintf("%d/%d", alreadyHeld, len(step.ReceivesKeyItems))
			}

			newSumm = QuestSummary{
				QuestID:    q.Quest.QuestID,
				Status:     statusString,
				Progress:   progress,
				QuestItems: questItemCount,
			}

		} else {
			newSumm = QuestSummary{
				QuestID: q.Quest.QuestID,
				Status:  statusString,
			}
		}

		questResponse = append(questResponse, newSumm)
	}

	player.PlayerMu.Unlock()

	rq, err := json.Marshal(questResponse)

	if err != nil {
		_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "err", err, "player", pname, "command", "QUESTS")

		return
	}

	_, _ = fmt.Fprintln(player.Conn, "OK "+string(rq))
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+string(rq), "command", "QUESTS")
}

// acceptQuest handles QUEST — adds a quest to player.Quests and sets it as Active.
func acceptQuest(player *Player, targetNPC string) error {
	pname := player.getPlayerName()
	npc, exists := resolveNPC(targetNPC)

	if !exists {
		return NPCNotFoundErr
	}

	key, quest, eligible := npc.getNPCQuest(player)

	if !eligible {
		return NoQuestAvailErr
	}

	player.PlayerMu.Lock()

	newQuest := PlayerQuest{
		Quest:     quest,
		StepIndex: 0,
		Status:    Active,
	}

	player.Quests[key] = &newQuest

	player.PlayerMu.Unlock()

	newQuestResponse := QuestResponse{
		QuestID:     quest.QuestID,
		Description: quest.Description,
		Reward:      quest.Reward,
		Status:      "available",
	}

	qr, err := json.Marshal(newQuestResponse)

	if err != nil {
		return JSONErr
	}

	_, _ = fmt.Fprintln(player.Conn, "OK "+string(qr))
	slog.Info("SYS_MESSAGE", "player", pname, "message", "OK "+string(qr), "command", "QUEST", "npc", targetNPC)

	return nil
}

// completeQuest marks the quest completed and grants its gold, key item and skill boost.
func (pq *PlayerQuest) completeQuest(player *Player) {
	pname := player.getPlayerName()

	var msg string

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if pq.Quest.Reward == nil || (pq.Quest.Reward.KeyItem == "" && pq.Quest.Reward.Money == 0) {
		msg = "Your kind deeds have been noted, but no reward is available."

		_, _ = fmt.Fprintln(player.Conn, msg)
		slog.Info("SYS_MESSAGE", "player", pname, "message", msg, "command", "completeQuest")

	} else {
		if pq.Quest.Reward.KeyItem != "" {
			i, ok := resolveKeyItem(pq.Quest.Reward.KeyItem)

			if !ok {
				msg = fmt.Sprintf("%s receives an imaginary key item.", pname)
				slog.Warn(ItemNotFoundErr.Error(), "key_item", pq.Quest.Reward.KeyItem, "quest", pq.Quest.Name, "player", pname)

			} else {
				player.KeyInventory[i.ItemID] = true

				if i.SkillBoost > 0 {
					player.BattleStats.BattleSkill += i.SkillBoost

					msg = fmt.Sprintf("%s receives %s.\nYou gain +%d BattleSkill", pname, i.ItemName, i.SkillBoost)

				} else {
					msg = fmt.Sprintf("%s receives %s.", pname, i.ItemName)
				}
			}

			_, _ = fmt.Fprintln(player.Conn, msg)
			slog.Info("SYS_MESSAGE", "player", pname, "message", msg, "command", "completeQuest")
		}

		if pq.Quest.Reward.Money > 0 {
			player.Gold += pq.Quest.Reward.Money
			msg = fmt.Sprintf("%s receives %d gold.", pname, pq.Quest.Reward.Money)

			_, _ = fmt.Fprintln(player.Conn, msg)
			slog.Info("SYS_MESSAGE", "player", pname, "message", msg, "command", "completeQuest")
		}
	}

	pq.Status = Completed
}

// --  Quest Actions -- //

// QuestAction is one step in a quest's Steps sequence.
type QuestAction interface {
	ActionType() string
}

// EnterAreaAction is a quest step completed by entering the named zone.
type EnterAreaAction struct {
	Area    string `yaml:"enter_area"`
	Message string `yaml:"message"`
}

// ActionType returns the step's discriminator key, "enter_area".
func (eaa *EnterAreaAction) ActionType() string { return "enter_area" }

// TalkToAction is a quest step completed by talking to the named NPC.
type TalkToAction struct {
	Target               string   `yaml:"talk_to"`
	Dialogue             string   `yaml:"dialogue"`
	GrantsKeyItems       []string `yaml:"grants_key_items"`
	ReceivesKeyItems     []string `yaml:"receives_key_items"`
	MissingItemsDialogue string   `yaml:"missing_items_dialogue"`
}

// ActionType returns the step's discriminator key, "talk_to".
func (tta *TalkToAction) ActionType() string { return "talk_to" }

// BattleAction is a quest step completed by defeating the named NPC.
type BattleAction struct {
	Target string `yaml:"battle"`
}

// ActionType returns the step's discriminator key, "battle".
func (ba *BattleAction) ActionType() string { return "battle" }
