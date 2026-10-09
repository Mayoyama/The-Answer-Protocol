package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type ReplyType int

const (
	ReplyIgnore ReplyType = iota
	ReplyLook
	ReplyPresence
	ReplyWho
	ReplyServerCount
	ReplyStatus
	ReplyQuest
	ReplyQuests
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
	ReplyOther
	ReplyError
)

type ReplyContent struct {
	Type        ReplyType
	Look        LookParse
	Who         WhoParse
	Status      StatusParse
	Presence    PresenceParse
	ServerCount int
	Quest       SingleQuestParse
	Quests      []QuestsParse
	Inventory   InventoryParse
	KeyItems    KeyItemParse
	TakeDrop    TakeDropItemParse
	Gold        int
	Examine     ExamineParse
	Group       GroupActionParse
	Chat        ChatParse
	Attack      AttackParse
	BattleFled  bool
	Respawned   string
	RoomID      string
	Err         ErrParse
}

type LookParse struct {
	Room    RoomParse `json:"room"`
	Players []string  `json:"players"`
	Items   []string  `json:"items"`
	NPCs    []string  `json:"npcs"`
}

type RoomParse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

type WhoParse struct {
	Room   []string `json:"room"`
	Server int      `json:"server"`
}

type StatusParse struct {
	HP     int    `json:"hp"`
	MaxHP  int    `json:"max_hp"`
	Status string `json:"status"`
}

type SingleQuestParse struct {
	QuestID     string      `json:"quest_id"`
	Description string      `json:"description"`
	Reward      RewardParse `json:"reward"`
	Status      string      `json:"status"`
}

type RewardParse struct {
	Gold    int    `json:"gold,omitempty"`
	KeyItem string `json:"key_item,omitempty"`
}

type QuestsParse struct {
	QuestID    string `json:"quest_id"`
	Status     string `json:"status"`
	Progress   string `json:"progress,omitempty"`
	QuestItems string `json:"quest_items"`
}

type InventoryParse []string

type KeyItemParse []string

type ExamineParse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Obtainable  bool   `json:"obtainable,omitempty"`
	NPCRole     string `json:"npc_role,omitempty"`
}

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

type ErrParse struct {
	Text string
	Code int
}

type TakeDropItemParse struct {
	Action string
	Item   string
}

type GroupActionParse struct {
	Action string
	byWho  string
}

type ChatParse struct {
	Scope   string
	Sender  string
	Message string
}

type PresenceParse struct {
	Direction string
	Who       string
}

func ParseServerReply(reply string) (ReplyContent, error) {
	toParse := strings.TrimSpace(reply)
	toUnmarshal := strings.TrimPrefix(toParse, "OK ")

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

	switch {
	case toParse == "OK []", toParse == "OK":
		return ReplyContent{Type: ReplyIgnore}, nil
	case strings.HasPrefix(toParse, "OK room="):
		roomID := strings.TrimPrefix(toParse, "OK room=")
		return ReplyContent{
			Type:   ReplyRoomMove,
			RoomID: roomID,
		}, nil

	case strings.HasPrefix(toParse, "OK {\"room\""):
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

	case strings.HasPrefix(toParse, "OK {\"hp\""):
		if err := json.Unmarshal([]byte(toUnmarshal), &status); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyStatus, Status: status}, nil

	case strings.HasPrefix(toParse, "OK {\"name\""):
		if err := json.Unmarshal([]byte(toUnmarshal), &examine); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyExamine, Examine: examine}, nil

	case strings.HasPrefix(toParse, "OK {\"attack"):
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

	case strings.HasPrefix(toParse, "OK group=PT-"):
		ldr := strings.TrimPrefix(toParse, "OK group=PT-")

		return ReplyContent{Type: ReplyGroup, Group: GroupActionParse{Action: "CREATE", byWho: ldr}}, nil

	case strings.HasPrefix(toParse, "EVT GROUP "):
		parts := strings.SplitN(toParse, " ", 4)

		if len(parts) < 4 {
			return ReplyContent{}, errors.New("malformed group event: " + toParse)
		}

		return ReplyContent{Type: ReplyGroup, Group: GroupActionParse{Action: parts[2], byWho: parts[3]}}, nil

	case strings.HasPrefix(toParse, "OK {\"quest_id\""):
		if err := json.Unmarshal([]byte(toUnmarshal), &quest); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyQuest, Quest: quest}, nil

	case strings.HasPrefix(toParse, "OK [{"):
		if err := json.Unmarshal([]byte(toUnmarshal), &quests); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyQuests, Quests: quests}, nil

	case strings.HasPrefix(toParse, "OK [\"item."):
		if err := json.Unmarshal([]byte(toUnmarshal), &inventory); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyInventory, Inventory: inventory}, nil

	case strings.HasPrefix(toParse, "OK [\"key_item."):
		if err := json.Unmarshal([]byte(toUnmarshal), &keyitems); err != nil {
			return ReplyContent{}, err
		}

		return ReplyContent{Type: ReplyKeyitems, KeyItems: keyitems}, nil

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
		return ReplyContent{Type: ReplyIgnore}, nil
	}
}
