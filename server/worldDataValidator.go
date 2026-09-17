package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// startingZone is the zone new players spawn into, and the root zone used for connectivity/cycle validation.
const startingZone = "taverne"

// validateMapConnectivity confirms every zone is reachable from startingZone via a breadth-first walk, returning an error if any zone is unreachable.
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

// mapLooper recursively walks currentZone's exits, returning true if it reaches a zone already on the current path (onPath).
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

// mapLoopExists reports whether the world's zone graph contains at least one cycle, walking from startingZone.
func mapLoopExists() bool {
	var (
		onPath        []string
		fullyExplored []string
	)

	onPath = append(onPath, startingZone)

	return mapLooper(startingZone, &onPath, &fullyExplored)
}

// ValidateWorldData checks zones, items, and NPCs (including their quests) for consistency, returning a list of errors.
func ValidateWorldData() []error {
	var (
		errs          []error
		questList     []*Quest
		existingItems = make(map[string]string)
	)

	if l := len(zones); l < 1 { // NEEDS TO BE 8
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_COUNT %d", l))
	}

	for _, z := range zones {
		if z.ZoneName == "" {
			errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_ROOM_NAME [nil]"))
		}

		if len(z.Exits) < 1 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NoExitErr.ErrorMessage, z.ZoneName))
		}

		if z.Description == "" {
			z.Description = "A mysterious area"
		}

		for dir, exit := range z.Exits {
			if dir == "" || strings.TrimSpace(dir) == "" {
				errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_DIRECTION [nil]"))
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
				obj.Description = "An item."
			}

			if _, exists := existingItems[itemID]; !exists {
				existingItems[itemID] = z.ZoneName

			} else {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: DUPLICATE_ITEM %s, LOCATION: %s", itemID, z.ZoneName))
			}
		}
	}

	for _, spawn := range npcs {
		if spawn.Description == "" {
			spawn.Description = "Anonymous"
		}

		if spawn.NPCName == "" || strings.TrimSpace(spawn.NPCName) == "" {
			errs = append(errs, errors.New("VALIDATION_ERROR: INVALID_NPC_NAME [nil]"))
		}

		switch spawn.Role {
		case QuestGiver:
			if len(spawn.Quests) == 0 {
				errs = append(errs, fmt.Errorf("VALIDATION_ERROR: QUESTGIVER_WITHOUT_QUESTS %s", spawn.NPCName))
			}

			if spawn.Attackable {
				errs = append(errs, errors.New("VALIDATION_ERROR: QUESTGIVER_CANNOT_BE_ATTACKABLE"))
			}

		case Enemy:
			if !spawn.Attackable {
				errs = append(errs, errors.New("VALIDATION_ERROR: ENEMY_NOT_ATTACKABLE"))
			}

			//if spawn HP is 0{
			//	errs = append(errs, errors.New("VALIDATION_ERROR: ATTACKABLE_NPC_HP0"))
			//}

		case General:
			if spawn.Attackable {
				errs = append(errs, errors.New("VALIDATION_ERROR: GENERAL_NPC_CANNOT_BE_ATTACKABLE"))
			}
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

			for _, action := range quest.Steps {
				switch val := action.(type) {
				case *TalkToAction:
					if _, ok := npcs[val.Target]; !ok {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NPCNotFoundErr.ErrorMessage, val.Target))
					}

				case *BattleAction:
					if n, ok := npcs[val.Target]; !ok {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NPCNotFoundErr.ErrorMessage, val.Target))

					} else {
						if !n.Attackable {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: %s %s", NPCNotHostileErr.ErrorMessage, val.Target))
						}

						if n.Role != Enemy {
							errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROLE %s", val.Target))
						}
					}

				case *EnterAreaAction:
					if _, ok := zones[val.Area]; !ok {
						errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_NAME %s", val.Area))
					}
				}
			}
		}
	}

	return errs
}
