need to decide if "direction" enum is needed in parsing YML and update both functions in parsing.go to reflect change



TalkToAction.Target and BattleAction.Target aren't checked against the npcs map — a quest step can point at an NPC that doesn't exist
EnterAreaAction.Area isn't checked against zones
BattleAction.Target isn't checked for being attackable/enemy-role, so a battle step could target an NPC that can't be fought


if npc is attackable but base HP is 0

if l := len(zones); l < 1 { // NEEDS TO BE 8
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_COUNT %d", l))
	}
