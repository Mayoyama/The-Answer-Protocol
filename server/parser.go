package main

import (
	"errors"
	yml "gopkg.in/yaml.v3"
)

type YmlData struct {
	World World
	Items map[string]*ParseItem
	NPCs  map[string]*ParseNPC
}

type World struct {
	Locations map[string]*ParseLoc
}

type ParseLoc struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Exits       map[string]string `yaml:"exits"`
	Items       []string          `yaml:"items"`
	Spawns      []string          `yaml:"spawns"`
}

type ParseItem struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Obtainable  bool   `yaml:"obtainable"`
}

type ParseNPC struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Dialogue    []string       `yaml:"dialogue"`
	Role        string         `yaml:"role"`
	Attackable  bool           `yaml:"attackable"`
	Stats       map[string]int `yaml:"stats"`
}

func ParseYmlData(data []byte) error {
	var worldData YmlData

	err := yml.Unmarshal(data, &worldData)
	if err != nil {
		return err
	}

	for key, item := range worldData.Items {
		itemName := item.Name
		newitem := Item{
			ItemID:      "item." + key,
			ItemName:    itemName,
			Description: item.Description,
			Obtainable:  item.Obtainable,
		}
		items[key] = &newitem
	}

	for key, npc := range worldData.NPCs {
		var npcRole NPCRole
		switch npc.Role {
		case "enemy", "Enemy":
			npcRole = Enemy
		case "quest giver", "Quest giver", "Quest Giver", "quest_giver":
			npcRole = QuestGiver
		default:
			npcRole = General
		}

		newNPC := NPC{
			NPCID:       "npc." + key,
			NPCName:     npc.Name,
			Description: npc.Description,
			Dialogue:    npc.Dialogue,
			Quests:      make(map[string]*Quest),
			Role:        npcRole,
			Attackable:  npc.Attackable,
			Stats:       npc.Stats,
		}
		npcs[key] = &newNPC
	}

	for key, zone := range worldData.World.Locations {
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
				return errors.New("PARSING ERROR: " + ItemNotFoundErr.Error() + " " + item + ", Zone: " + newLoc.ZoneID)
			}

			newLoc.Items[nm.ItemID] = nm
			nm.BaseLoc = &newLoc

		}

		for _, npc := range zone.Spawns {
			n, ok := resolveNPC(npc)
			if !ok {
				return errors.New("PARSING ERROR: " + NPCNotFoundErr.Error() + " " + npc + ", Zone: " + newLoc.ZoneID)
			}

			newLoc.NPCs[n.NPCID] = n
			n.BaseLoc = &newLoc
		}

		zones[key] = &newLoc
	}

	return nil
}

func ValidateWorldData() error {

	return nil
}
