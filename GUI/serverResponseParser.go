package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// ReplyType identifies the kind of server line parseServerReply found.
type ReplyType int

// ReplyType values returned by parseServerReply.
const (
	ReplyIgnore ReplyType = iota
	ReplyEmptyList
	ReplyLook
	ReplyPresence
	ReplyWho
	ReplyServerCount
	ReplyStatus
	ReplyQuest
	ReplyQuests
	ReplyReward
	ReplyTakeDrop
	ReplyInventory
	ReplyKeyitems
	ReplyGold
	ReplyGroup
	ReplyExamine
	ReplyAttack
	ReplyBattleEnd
	ReplyRespawned
	ReplyRoomMove
	ReplyChat
	ReplyNPCDialogue
	ReplyOther
	ReplyError
)

// ReplyContent is a parsed server line; Type says which field holds its data.
type ReplyContent struct {
	Type           ReplyType
	Look           LookParse
	Who            WhoParse
	Status         StatusParse
	Presence       PresenceParse
	ServerCount    int
	Quest          SingleQuestParse
	Quests         []QuestsParse
	Rewards        RewardParse
	RewardReceiver string
	Inventory      InventoryParse
	KeyItems       KeyItemParse
	TakeDrop       TakeDropItemParse
	Gold           int
	Examine        ExamineParse
	Group          GroupActionParse
	Chat           ChatParse
	Attack         AttackParse
	BattleFled     bool
	Respawned      string
	RoomID         string
	NPCName        string
	NPCText        string
	NPCHeal        bool
	Err            ErrParse
	Other          string
}

// LookParse is the LOOK reply.
type LookParse struct {
	Room    RoomParse `json:"room"`
	Players []string  `json:"players"`
	Items   []string  `json:"items"`
	NPCs    []string  `json:"npcs"`
}

// RoomParse is the room part of a LOOK reply.
type RoomParse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

// WhoParse is the WHO reply, in the subject PDF's JSON form.
type WhoParse struct {
	Room   []string `json:"room"`
	Server int      `json:"server"`
}

// StatusParse is the STATUS reply.
type StatusParse struct {
	HP     int    `json:"hp"`
	MaxHP  int    `json:"max_hp"`
	Status string `json:"status"`
}

// SingleQuestParse is the QUEST reply.
type SingleQuestParse struct {
	QuestID     string      `json:"quest_id"`
	Description string      `json:"description"`
	Reward      RewardParse `json:"reward"`
	Status      string      `json:"status"`
}

// RewardParse is a quest reward: gold, a key item, or both.
type RewardParse struct {
	Gold    int    `json:"gold,omitempty"`
	KeyItem string `json:"key_item,omitempty"`
}

// QuestsParse is one entry of the QUESTS reply.
type QuestsParse struct {
	QuestID    string `json:"quest_id"`
	Status     string `json:"status"`
	Progress   string `json:"progress,omitempty"`
	QuestItems string `json:"quest_items"`
}

// InventoryParse is the INVENTORY reply: a list of item IDs.
type InventoryParse []string

// KeyItemParse is the KEYITEMS reply: a list of key item IDs.
type KeyItemParse []string

// ExamineParse is the EXAMINE reply.
type ExamineParse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Obtainable  bool   `json:"obtainable,omitempty"`
	NPCRole     string `json:"npc_role,omitempty"`
}

// AttackParse is the reply to one ATTACK round.
type AttackParse struct {
	PlayerInitRoll int    `json:"attacker_init_roll"`
	TargetInitRoll int    `json:"target_init_roll"`
	PlayerHP       int    `json:"attacker_hp"`
	PlayerMaxHP    int    `json:"attacker_max_hp"`
	TargetHP       int    `json:"target_hp"`
	TargetMaxHP    int    `json:"target_max_hp"`
	TargetDodged   bool   `json:"target_dodged"`
	TargetDmgRecvd int    `json:"damage"`
	PlayerDodged   bool   `json:"attacker_dodged"`
	PlayerDmgRecvd int    `json:"damage_received"`
	PlayerStatus   string `json:"status"`
}

// ErrParse is an ERR line's code and text.
type ErrParse struct {
	Text string
	Code int
}

// TakeDropItemParse is an "OK taken=" or "OK dropped=" reply.
type TakeDropItemParse struct {
	Action string
	Item   string
}

// GroupActionParse is a group reply or event: the action and the player it names.
type GroupActionParse struct {
	Action string
	ByWho  string
}

// ChatParse is a chat event: scope, sender and message.
type ChatParse struct {
	Scope   string
	Sender  string
	Message string
}

// TalkParse is the JSON form of the TALK reply.
type TalkParse struct {
	NPC      string `json:"npc"`
	Dialogue string `json:"dialogue"`
}

// PresenceParse is a room presence event: ENTER or LEAVE, and the player.
type PresenceParse struct {
	Direction string
	Who       string
}

// parseServerReply turns one server line into a ReplyContent. Lines it doesn't
// recognise become ReplyOther, holding the raw line.
func parseServerReply(reply string) (ReplyContent, error) {
	toParse := strings.TrimSpace(reply)
	toUnmarshal := strings.TrimSpace(strings.TrimPrefix(toParse, "OK "))

	var (
		look      LookParse
		who       WhoParse
		status    StatusParse
		quest     SingleQuestParse
		quests    []QuestsParse
		inventory InventoryParse
		keyitems  KeyItemParse
		examine   ExamineParse
		attack    AttackParse
	)

	if receiver, amount, ok := isGoldReward(toParse); ok {
		return ReplyContent{Type: ReplyReward, Rewards: RewardParse{Gold: amount}, RewardReceiver: receiver}, nil
	}

	if receiver, keyitem, ok := isKeyItemReward(toParse); ok {
		return ReplyContent{Type: ReplyReward, Rewards: RewardParse{KeyItem: keyitem}, RewardReceiver: receiver}, nil
	}

	switch {
	case toParse == "OK":
		return ReplyContent{Type: ReplyIgnore}, nil

	case toParse != "OK [" && strings.TrimSpace(strings.TrimSuffix(toParse, "]")) == "OK [":
		return ReplyContent{Type: ReplyEmptyList}, nil

	case strings.HasPrefix(toParse, "OK room="):
		roomID := strings.TrimPrefix(toParse, "OK room=")

		return ReplyContent{
			Type:   ReplyRoomMove,
			RoomID: roomID,
		}, nil

	case strings.HasPrefix(toParse, "OK {\"room\"") || strings.HasPrefix(toParse, "OK { \"room\""):
		if strings.Contains(toParse, "\"server\"") {
			if err := json.Unmarshal([]byte(toUnmarshal), &who); err != nil {
				return ReplyContent{}, err
			}

			return ReplyContent{Type: ReplyWho, Who: who}, nil

		} else {
			if err := json.Unmarshal([]byte(toUnmarshal), &look); err != nil {
				return ReplyContent{}, err
			}

			return ReplyContent{Type: ReplyLook, Look: look}, nil
		}

	case strings.HasPrefix(toParse, "OK {\"hp\"") || strings.HasPrefix(toParse, "OK { \"hp\""):
		if err := json.Unmarshal([]byte(toUnmarshal), &status); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyStatus, Status: status}, nil

	case strings.HasPrefix(toParse, "OK {\"name\"") || strings.HasPrefix(toParse, "OK { \"name\""):
		if err := json.Unmarshal([]byte(toUnmarshal), &examine); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyExamine, Examine: examine}, nil

	case strings.HasPrefix(toParse, "OK {\"attack") || strings.HasPrefix(toParse, "OK { \"attack"):
		if err := json.Unmarshal([]byte(toUnmarshal), &attack); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyAttack, Attack: attack}, nil

	case toParse == "OK battle ended":
		return ReplyContent{Type: ReplyBattleEnd, BattleFled: true}, nil

	case strings.HasPrefix(toParse, "OK ") && len(strings.Fields(toParse)) == 3 && strings.HasSuffix(toParse, "GOLD"):
		toAtoi := strings.Fields(toParse)[1]
		num, err := strconv.Atoi(toAtoi)

		if err != nil {
			return ReplyContent{Type: ReplyIgnore}, nil
		}

		return ReplyContent{Type: ReplyGold, Gold: num}, nil

	case strings.HasPrefix(toParse, "EVT RESPAWN room="):
		return ReplyContent{Type: ReplyRespawned, Respawned: strings.TrimPrefix(toParse, "EVT RESPAWN room=")}, nil

	case strings.HasPrefix(toParse, "OK taken="), strings.HasPrefix(toParse, "OK dropped="):
		parts := strings.SplitN(toUnmarshal, "=", 2)

		if len(parts) != 2 {
			return ReplyContent{}, errors.New("malformed take/drop event: " + toParse)
		}

		tdAction := TakeDropItemParse{
			Action: parts[0],
			Item:   parts[1],
		}

		return ReplyContent{Type: ReplyTakeDrop, TakeDrop: tdAction}, nil

	case strings.HasPrefix(toParse, "EVT ROOM PRESENCE ENTER"), strings.HasPrefix(toParse, "EVT ROOM PRESENCE LEAVE"):
		parts := strings.Fields(toParse)

		if len(parts) != 5 {
			return ReplyContent{}, errors.New("malformed presence event: " + toParse)
		}

		return ReplyContent{Type: ReplyPresence, Presence: PresenceParse{Direction: parts[3], Who: parts[4]}}, nil

	case strings.HasPrefix(toParse, "EVT STATS players="):
		parts := strings.SplitN(toUnmarshal, "=", 2)

		if len(parts) != 2 {
			return ReplyContent{}, errors.New("malformed evt stats event: " + toParse)
		}

		count, err := strconv.Atoi(parts[1])

		if err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyServerCount, ServerCount: count}, nil

	case strings.HasPrefix(toParse, "EVT GROUP CHAT"), strings.HasPrefix(toParse, "EVT ROOM CHAT"), strings.HasPrefix(toParse, "EVT GLOBAL CHAT"):
		var msg string

		parts := strings.SplitN(toParse, " ", 5)

		if len(parts) < 4 {
			return ReplyContent{}, errors.New("malformed chat event: " + toParse)
		}

		if len(parts) > 4 {
			msg = parts[4]
		}

		return ReplyContent{Type: ReplyChat, Chat: ChatParse{Scope: parts[1], Sender: parts[3], Message: msg}}, nil

	case strings.HasPrefix(toParse, "OK group=NEWPT-"):
		ldr := strings.TrimPrefix(toParse, "OK group=NEWPT-")

		return ReplyContent{Type: ReplyGroup, Group: GroupActionParse{Action: "CREATE", ByWho: ldr}}, nil

	case strings.HasPrefix(toParse, "OK group=PT-"):
		ldr := strings.TrimPrefix(toParse, "OK group=PT-")

		return ReplyContent{Type: ReplyGroup, Group: GroupActionParse{Action: "JOIN", ByWho: ldr}}, nil

	case strings.HasPrefix(toParse, "EVT GROUP "):
		parts := strings.SplitN(toParse, " ", 4)

		if len(parts) < 4 {
			return ReplyContent{}, errors.New("malformed group event: " + toParse)
		}

		return ReplyContent{Type: ReplyGroup, Group: GroupActionParse{Action: parts[2], ByWho: parts[3]}}, nil

	case strings.HasPrefix(toParse, "OK {\"quest_id\"") || strings.HasPrefix(toParse, "OK { \"quest_id\""):
		if err := json.Unmarshal([]byte(toUnmarshal), &quest); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyQuest, Quest: quest}, nil

	case strings.HasPrefix(toParse, "OK [{") || strings.HasPrefix(toParse, "OK [ {"):
		if err := json.Unmarshal([]byte(toUnmarshal), &quests); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyQuests, Quests: quests}, nil

	case strings.HasPrefix(toParse, "OK [\"item.") || strings.HasPrefix(toParse, "OK [ \"item."):
		if err := json.Unmarshal([]byte(toUnmarshal), &inventory); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyInventory, Inventory: inventory}, nil

	case strings.HasPrefix(toParse, "OK [\"key_item.") || strings.HasPrefix(toParse, "OK [ \"key_item."):
		if err := json.Unmarshal([]byte(toUnmarshal), &keyitems); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyKeyitems, KeyItems: keyitems}, nil

	case strings.HasPrefix(toParse, "OK "):
		var (
			talk TalkParse
			name string
		)

		npcText := toUnmarshal

		if strings.HasPrefix(toUnmarshal, "{") {
			if err := json.Unmarshal([]byte(toUnmarshal), &talk); err == nil && talk.Dialogue != "" {
				name = talk.NPC
				npcText = talk.Dialogue
			}
		}

		isHeal := strings.Contains(npcText, "(You feel a warm glow)")

		return ReplyContent{Type: ReplyNPCDialogue, NPCText: npcText, NPCName: name, NPCHeal: isHeal}, nil

	case strings.HasPrefix(toParse, "ERR "):
		parts := strings.SplitN(toParse, " ", 3)

		if len(parts) < 3 {
			return ReplyContent{}, errors.New("malformed error event: " + toParse)
		}

		errCode, err := strconv.Atoi(parts[1])

		if err != nil {
			return ReplyContent{}, err
		}

		errInfo := ErrParse{
			Text: parts[2],
			Code: errCode,
		}

		return ReplyContent{Type: ReplyError, Err: errInfo}, nil

	default:
		return ReplyContent{Type: ReplyOther, Other: toParse}, nil
	}
}

// isGoldReward matches "<name> receives <n> gold." and returns the name and amount.
func isGoldReward(line string) (string, int, bool) {
	parts := strings.Fields(line)

	if len(parts) != 4 {
		return "", 0, false
	}

	if parts[1] != "receives" || parts[3] != "gold." {
		return "", 0, false
	}

	amount, err := strconv.Atoi(parts[2])

	if err != nil {
		return "", 0, false
	}

	return parts[0], amount, true
}

// isKeyItemReward matches "<name> receives <item>." and returns the name and item.
func isKeyItemReward(line string) (string, string, bool) {
	noDot := strings.TrimSuffix(line, ".")
	parts := strings.Fields(noDot)

	if len(parts) < 3 {
		return "", "", false
	}

	if parts[1] != "receives" {
		return "", "", false
	}

	if !strings.HasSuffix(line, ".") {
		return "", "", false
	}

	rewardItem := strings.Join(parts[2:], " ")

	return parts[0], rewardItem, true
}
