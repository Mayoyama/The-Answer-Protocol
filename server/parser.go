package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

// YmlData is the top-level structure of world.yaml.
type YmlData struct {
	World  World
	Items  map[string]*ParseItem
	NPCs   map[string]*ParseNPC
	Quests map[string]*ParseQuest
}

// World holds all locations defined in world.yaml.
type World struct {
	Locations map[string]*ParseLoc
}

// ParseLoc is a location entry as read from world.yaml.
type ParseLoc struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Exits       map[string]string `yaml:"exits"`
	Items       []string          `yaml:"items"`
	Spawns      []string          `yaml:"spawns"`
}

// ParseItem is an item entry as read from world.yaml.
type ParseItem struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Obtainable  bool   `yaml:"obtainable"`
}

// ParseNPC is an NPC entry as read from world.yaml.
type ParseNPC struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Dialogue    []string       `yaml:"dialogue"`
	Role        string         `yaml:"role"`
	Attackable  bool           `yaml:"attackable"`
	Stats       map[string]int `yaml:"stats"`
	Quests      []string       `yaml:"quests"`
}

// ParseQuest is a quest entry as read from world.yaml.
type ParseQuest struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Type        string       `yaml:"type"`
	Steps       []*QuestStep `yaml:"steps"`
	Reward      *Reward      `yaml:"reward"`
}

// QuestStep wraps a single quest step, decoded into its concrete QuestAction via UnmarshalYAML.
type QuestStep struct {
	Action QuestAction
}

// UnmarshalYAML decodes a quest step into the QuestAction matching its discriminator key (talk_to, enter_area, or battle).
func (qs *QuestStep) UnmarshalYAML(value *yaml.Node) error {
	var rawData map[string]any

	if err := value.Decode(&rawData); err != nil {
		return err
	}

	switch {
	case rawData["talk_to"] != nil:
		var action TalkToAction

		if err := value.Decode(&action); err != nil {
			return err
		}

		qs.Action = &action

	case rawData["enter_area"] != nil:
		var action EnterAreaAction

		if err := value.Decode(&action); err != nil {
			return err
		}

		qs.Action = &action

	case rawData["battle"] != nil:
		var action BattleAction

		if err := value.Decode(&action); err != nil {
			return err
		}

		qs.Action = &action

	default:
		return fmt.Errorf("PARSING ERROR: INVALID_STEPS_TYPE")
	}

	return nil
}

// ParseYmlData loads world.yaml into the items, npcs, and zones maps.
func ParseYmlData(data []byte) error {
	var worldData YmlData

	err := yaml.Unmarshal(data, &worldData)

	if err != nil {
		return err
	}

	for key, item := range worldData.Items {
		if key == "" || strings.TrimSpace(key) == "" {
			return errors.New("PARSING_ERROR: INVALID_ITEM_KEY [nil]")
		}

		newitem := Item{
			ItemID:      "item." + key,
			ItemName:    item.Name,
			Description: item.Description,
			Obtainable:  item.Obtainable,
		}

		items[key] = &newitem
	}

	var questObjList = make(map[string]*Quest)

	for key, quest := range worldData.Quests {
		if key == "" || strings.TrimSpace(key) == "" {
			return errors.New("PARSING_ERROR: INVALID_QUEST_KEY [nil]")
		}

		var questType QuestType

		switch quest.Type {
		case "delivery", "Delivery":
			questType = Delivery

		case "fetch", "Fetch":
			questType = Fetch

		case "Battle", "battle", "fight", "Fight":
			questType = Battle

		default:
			return fmt.Errorf("PARSING_ERROR: INVALID_QUEST_TYPE %s, QUEST_NAME: %s", quest.Type, quest.Name)
		}

		var steplist []QuestAction

		for _, step := range quest.Steps {
			steplist = append(steplist, step.Action)
		}

		newQuest := Quest{
			Name:        quest.Name,
			QuestID:     "quest." + key,
			Description: quest.Description,
			Type:        questType,
			Steps:       steplist,
			Reward:      quest.Reward,
		}

		questObjList[key] = &newQuest
	}

	for key, npc := range worldData.NPCs {
		if key == "" || strings.TrimSpace(key) == "" {
			return errors.New("PARSING_ERROR: INVALID_NPC_KEY [nil]")
		}

		var npcRole NPCRole

		switch npc.Role {
		case "enemy", "Enemy":
			npcRole = Enemy

		case "quest giver", "Quest giver", "Quest Giver", "quest_giver":
			npcRole = QuestGiver

		case "general", "General":
			npcRole = General

		default:
			return fmt.Errorf("PARSING_ERROR: INVALID_NPC_ROLE %s, ROLE: %s", npc.Name, npc.Role)
		}

		var questList = make(map[string]*Quest)

		if len(npc.Quests) > 0 {
			for _, q := range npc.Quests {
				task, ok := questObjList[q]

				if !ok {
					return fmt.Errorf("PARSING_ERROR: INVALID_NPC_QUEST_NAME %s, NPC: %s", q, npc.Name)
				}

				questList[q] = task
			}
		}

		newNPC := NPC{
			NPCID:       "npc." + key,
			NPCName:     npc.Name,
			Description: npc.Description,
			Dialogue:    npc.Dialogue,
			Quests:      questList,
			Role:        npcRole,
			Attackable:  npc.Attackable,
			Stats:       npc.Stats,
		}

		npcs[key] = &newNPC
	}

	for key, zone := range worldData.World.Locations {
		if key == "" || strings.TrimSpace(key) == "" {
			return errors.New("PARSING_ERROR: INVALID_LOCATION_KEY [nil]")
		}

		newLoc := Zone{
			ZoneID:      "zone." + key,
			ZoneName:    zone.Name,
			Description: zone.Description,
			Exits:       zone.Exits,
			InZone:      make(map[string]*Player),
			Items:       make(map[string]*Item),
			NPCs:        make(map[string]*NPC),
		}

		for _, item := range zone.Items {
			nm, ok := resolveItem(item)

			if !ok {
				return fmt.Errorf("PARSING_ERROR: %s %s, LOCATION: %s", ItemNotFoundErr.ErrorMessage, item, newLoc.ZoneID)
			}

			newLoc.Items[nm.ItemID] = nm
			nm.BaseLoc = &newLoc

		}

		for _, npc := range zone.Spawns {
			n, ok := resolveNPC(npc)
			if !ok {
				return fmt.Errorf("PARSING_ERROR: %s %s, LOCATION: %s", NPCNotFoundErr.ErrorMessage, npc, newLoc.ZoneID)
			}

			newLoc.NPCs[n.NPCID] = n
			n.BaseLoc = &newLoc
		}

		zones[key] = &newLoc
	}

	return nil
}
