package main

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateWorldData checks the loaded world data for consistency. Not yet implemented.
func ValidateWorldData() []error {
	var errs []error

	existingItems := make(map[string]string)

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

		if spawn.Role == QuestGiver && len(spawn.Quests) == 0 {
			errs = append(errs, fmt.Errorf("VALIDATION_ERROR: QUESTGIVER_WITHOUT_QUESTS %s", spawn.NPCName))
		}
	}

	return errs
}
