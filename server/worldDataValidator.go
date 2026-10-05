package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// startingZone is the zone new players spawn into, and the root zone used for
// connectivity/cycle validation.
// respawnZone is where players respawn after losing a fight.
const (
	startingZone = "taverne"
	respawnZone  = "chapel"
)

// validateMapConnectivity confirms every zone is reachable from startingZone via a
// breadth-first walk, returning an error if any zone is unreachable.
func validateMapConnectivity() error {
	var (
		visited []string
		queue   []string
	)

	queue = append(queue, startingZone)

	for len(queue) >= 1 {
		currentZone := queue[0]
		visited = append(visited, currentZone)
		queue = queue[1:]

		for _, zConn := range zones[currentZone].Exits {
			if !slices.Contains(visited, zConn) && !slices.Contains(queue, zConn) {
				queue = append(queue, zConn)
			}
		}
	}

	lv := len(visited)
	lz := len(zones)

	if lv != lz {
		return fmt.Errorf("VALIDATION_ERROR: CONNECTED_ZONES %d, EXPECTED %d", lv, lz)
	}

	return nil
}

// mapLooper recursively walks currentZone's exits, returning true if it reaches a zone
// already on the current path (onPath).
func mapLooper(currentZone string, onPath, fullyExplored *[]string) bool {
	for _, z := range zones[currentZone].Exits {
		if slices.Contains(*onPath, z) {
			return true
		}

		if slices.Contains(*fullyExplored, z) {
			continue
		}

		*onPath = append(*onPath, z)

		if mapLooper(z, onPath, fullyExplored) {
			return true
		}

		*fullyExplored = append(*fullyExplored, z)
		*onPath = slices.DeleteFunc(*onPath, func(s string) bool { return s == z })

	}

	return false
}

// mapLoopExists reports whether the world's zone graph contains at least one cycle,
// walking from startingZone.
func mapLoopExists() bool {
	var (
		onPath        []string
		fullyExplored []string
	)

	onPath = append(onPath, startingZone)

	return mapLooper(startingZone, &onPath, &fullyExplored)
}

// ValidateWorldData checks zones, items, keyItems and NPCs (including their quests) for
// consistency, returning a list of errors.
func ValidateWorldData() []error {
	var (
		errs                 []error
		questList            []*Quest
		existingItems        = make(map[string]string)
		obtainableItems      []string
		ItemKIComparisonList []string
		grantedKeyItems      = make(map[string]bool)
		roleList             []string
	)

	if _, startZoneOK := zones[startingZone]; !startZoneOK {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_STARTING_ROOM %s", startingZone))
	}

	if _, respawnZoneOK := zones[respawnZone]; !respawnZoneOK {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_RESPAWN_ROOM %s", respawnZone))
	}

	if l := len(zones); l < 8 {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_COUNT %d", l))
	}

	for _, z := range zones {
		if z.ZoneName == "" || strings.TrimSpace(z.ZoneName) == "" {
			errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_ROOM_NAME [nil]"))
		}

		if len(z.Exits) < 1 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NoExitErr.ErrorMessage, z.ZoneName))
		}

		if z.Description == "" || strings.TrimSpace(z.Description) == "" {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_DESCRIPTION [nil] %s", z.ZoneName))
		}

		for dir, exit := range z.Exits {
			if dir == InvalidDirection {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_DIRECTION, ROOM: %s", z.ZoneName))
			}

			if exit == "" || strings.TrimSpace(exit) == "" {
				errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_EXIT [nil]"))

			} else if _, ok := zones[exit]; !ok {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_EXIT %s", exit))
			}
		}

		for itemID, obj := range z.Items {
			if obj.ItemName == "" || strings.TrimSpace(obj.ItemName) == "" {
				errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_ITEM_NAME [nil]"))
			}

			if obj.Description == "" || strings.TrimSpace(obj.Description) == "" {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ITEM_DESCRIPTION [nil] %s", obj.ItemName))
			}

			if _, exists := existingItems[itemID]; !exists {
				existingItems[itemID] = z.ZoneName

				if slices.Contains(ItemKIComparisonList, strings.ToLower(obj.ItemName)) {
					errs = append(errs, fmt.Errorf("VALIDATION_ERROR: DUPLICATE_ITEM_NAME %s, LOCATION: %s", strings.ToLower(obj.ItemName), z.ZoneName))

				} else {
					ItemKIComparisonList = append(ItemKIComparisonList, strings.ToLower(obj.ItemName))
				}

				if obj.Obtainable {
					obtainableItems = append(obtainableItems, itemID)
				}

			} else {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: DUPLICATE_ITEM_ID %s, LOCATION: %s", itemID, z.ZoneName))
			}
		}
	}

	for _, n := range npcs {
		for _, q := range n.Quests {
			for _, action := range q.Steps {
				step, isTTA := action.(*TalkToAction)

				if !isTTA {
					continue
				}

				for _, ki := range step.GrantsKeyItems {
					gki, exists := resolveKeyItem(ki)

					if exists {
						grantedKeyItems[gki.ItemID] = true
					}
				}
			}

			if q.Reward != nil && q.Reward.KeyItem != "" {
				if reward, ok := resolveKeyItem(q.Reward.KeyItem); ok {
					grantedKeyItems[reward.ItemID] = true
				}
			}
		}
	}

	for _, spawn := range npcs {
		if spawn.NPCName == "" || strings.TrimSpace(spawn.NPCName) == "" {
			errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_NPC_NAME [nil]"))
		}

		if spawn.Description == "" || strings.TrimSpace(spawn.Description) == "" {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_NPC_DESCRIPTION [nil] %s", spawn.NPCName))
		}

		if spawn.Stats.HP <= 0 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_NPC_HP %s", spawn.NPCName))
		}

		if spawn.Stats.Strength < 0 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NEGATIVE_NPC_STR %s", spawn.NPCName))
		}

		if spawn.Stats.BattleSkill < 0 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NEGATIVE_NPC_BATTLESKILL %s", spawn.NPCName))
		}

		if spawn.Stats.Dexterity < 0 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NEGATIVE_NPC_DEX %s", spawn.NPCName))
		}

		switch spawn.Role {
		case QuestGiver:
			if len(spawn.Quests) == 0 {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: QUESTGIVER_WITHOUT_QUESTS %s", spawn.NPCName))
			}

			if spawn.Attackable {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: QUESTGIVER_CANNOT_BE_ATTACKABLE %s", spawn.NPCName))
			}

			if !slices.Contains(roleList, "QuestGiver") {
				roleList = append(roleList, "QuestGiver")
			}

		case Enemy:
			if !spawn.Attackable {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: ENEMY_NOT_ATTACKABLE %s", spawn.NPCName))
			}

			if spawn.Stats.BattleSkill < 1 || spawn.Stats.Strength < 1 {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: ATTACKABLE_NPC_WITH_INSUFFICIENT_STATS %s", spawn.NPCName))
			}

			if spawn.Stats.Dexterity > 90 {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: ATTACKABLE_NPC_DEX_EXCEEDS_LIMITS %s", spawn.NPCName))
			}

			if spawn.BattleDialogue.BattleStart == "" || strings.TrimSpace(spawn.BattleDialogue.BattleStart) == "" {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NPC_BATTLESTART_DIALOGUE [nil] %s", spawn.NPCName))
			}

			if spawn.BattleDialogue.PlayerVictory == "" || strings.TrimSpace(spawn.BattleDialogue.PlayerVictory) == "" {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NPC_PLAYERVICTORY_DIALOGUE [nil] %s", spawn.NPCName))
			}

			if spawn.BattleDialogue.NPCVictory == "" || strings.TrimSpace(spawn.BattleDialogue.NPCVictory) == "" {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NPC_NPCVICTORY_DIALOGUE [nil] %s", spawn.NPCName))
			}

			if !slices.Contains(roleList, "Enemy") {
				roleList = append(roleList, "Enemy")
			}

		case Healer:
			if spawn.Attackable {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: HEALER_CANNOT_BE_ATTACKABLE %s", spawn.NPCName))
			}

			if spawn.HealDialogue == "" || strings.TrimSpace(spawn.HealDialogue) == "" {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NPC_HEALER_HEALING_DIALOGUE [nil] %s", spawn.NPCName))
			}

			if !slices.Contains(roleList, "Healer") {
				roleList = append(roleList, "Healer")
			}

		case General:
			if spawn.Attackable {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: GENERAL_NPC_CANNOT_BE_ATTACKABLE %s", spawn.NPCName))
			}

			if !slices.Contains(roleList, "General") {
				roleList = append(roleList, "General")
			}
		default:
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_NPC_ROLE %s", spawn.NPCName))
		}

		for _, quest := range spawn.Quests {
			if quest.Description == "" || strings.TrimSpace(quest.Description) == "" {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_QUEST_DESCRIPTION [nil], QUEST: %s", quest.Name))
			}

			if slices.ContainsFunc(questList, func(q *Quest) bool {
				return quest.Name == q.Name || strings.TrimPrefix(quest.QuestID, "quest.") == strings.TrimPrefix(q.QuestID, "quest.")
			}) {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: DUPLICATE_QUEST_FOUND %s", quest.Name))

			} else {
				questList = append(questList, quest)
			}

			if quest.Reward != nil && quest.Reward.KeyItem != "" {
				_, exists := resolveKeyItem(quest.Reward.KeyItem)

				if !exists {
					errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s, QUEST: %s", ItemNotFoundErr.ErrorMessage, quest.Reward.KeyItem, quest.Name))
				}
			}

			var sameQuestKeyItems []string

			for _, action := range quest.Steps {
				switch val := action.(type) {
				case *TalkToAction:
					if n, ok := npcs[val.Target]; !ok {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NPCNotFoundErr.ErrorMessage, val.Target))

					} else {
						if n.Role == Healer {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROLE_FOR_QUEST_ACTION %s, ROLE: healer", val.Target))
						}
					}

					if val.MissingItemsDialogue != "" && strings.TrimSpace(val.MissingItemsDialogue) == "" {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_MISSING_ITEMS_DIALOGUE [%s], QUEST: %s", val.MissingItemsDialogue, quest.Name))
					}

					for _, ki := range val.GrantsKeyItems {
						gki, exists := resolveKeyItem(ki)

						if !exists {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", ItemNotFoundErr.ErrorMessage, ki))

							continue
						}

						if slices.Contains(sameQuestKeyItems, gki.ItemID) {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: DUPLICATE_KEY_ITEM_FOUND %s", ki))

						} else {
							sameQuestKeyItems = append(sameQuestKeyItems, gki.ItemID)
						}
					}

					for _, ki := range val.ReceivesKeyItems {
						gki, exists := resolveKeyItem(ki)

						if !exists {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", ItemNotFoundErr.ErrorMessage, ki))

							continue
						}

						resIdx := slices.Index(sameQuestKeyItems, gki.ItemID)

						if resIdx == -1 {
							_, found := grantedKeyItems[gki.ItemID]

							if !found {
								errs = append(errs, fmt.Errorf("VALIDATION_ERROR: MISSING_QUEST_KEY_ITEM %s, QUEST: %s", ki, quest.Name))
							}

						} else {
							sameQuestKeyItems = slices.Delete(sameQuestKeyItems, resIdx, resIdx+1)
						}
					}

				case *BattleAction:
					if n, ok := npcs[val.Target]; !ok {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NPCNotFoundErr.ErrorMessage, val.Target))

					} else {
						if !n.Attackable {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NPCNotHostileErr.ErrorMessage, val.Target))
						}

						if n.Role != Enemy {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROLE_FOR_BATTLE_ACTION %s, ROLE: %s", val.Target, n.getNPCRole()))
						}
					}

				case *EnterAreaAction:
					_, ok := zones[val.Area]

					if !ok {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_NAME_FOR_ENTER_AREA_ACTION %s", val.Area))
					}

					if val.Message == "" || strings.TrimSpace(val.Message) == "" {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ENTER_AREA_ACTION_MESSAGE [nil], AREA: %s", val.Area))
					}
				}
			}
			if len(sameQuestKeyItems) > 0 {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: QUEST_KEY_ITEMS_NOT_BALANCED %v, QUEST: %s", sameQuestKeyItems, quest.Name))
			}
		}
	}

	for k, ki := range keyItems {
		if k == "" || ki.ItemName == "" || strings.TrimSpace(k) == "" || strings.TrimSpace(ki.ItemName) == "" {
			errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_KEY_ITEM_NAME [nil]"))
		}

		if ki.Description == "" || strings.TrimSpace(ki.Description) == "" {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_KEY_ITEM_DESCRIPTION [nil] %s", ki.ItemName))
		}

		if ki.SkillBoost < 0 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: NEGATIVE_KEY_ITEM_SKILLBOOST %d, KEY_ITEM: %s", ki.SkillBoost, ki.ItemName))
		}

		iname := strings.ToLower(ki.ItemName)
		if slices.Contains(ItemKIComparisonList, iname) {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: DUPLICATE_ITEM_NAME %s", iname))

		} else {
			ItemKIComparisonList = append(ItemKIComparisonList, iname)
		}
	}

	itemLen := len(items)
	if itemLen < 4 {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INSUFFICIENT_ITEM_COUNT %d/4", itemLen))
	}

	obtainableItemsLen := len(obtainableItems)
	if obtainableItemsLen < 2 {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INSUFFICIENT_OBTAINABLE_ITEM_COUNT %d/2", obtainableItemsLen))
	}

	roleListLen := len(roleList)
	if roleListLen < 3 {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INSUFFICIENT_NPC_ROLE_COUNT %d/3", roleListLen))
	}

	questLen := len(questList)
	if questLen < 2 {
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INSUFFICIENT_QUEST_COUNT %d/2", questLen))
	}

	return errs
}
