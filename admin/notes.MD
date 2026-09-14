need to decide if "direction" enum is needed in parsing YML and update both functions in parsing.go to reflect change

need BFS/DFS to confirm connectivity and another algo to check the loop

need to improve NPC validation as quests/battle/stats are improved



On the soft convention (capitalize "concept" types/constructors, lowercase narrow internal helpers), the rest of the codebase is actually pretty consistent — all your get* getters, handle* handlers, print*, commandDispatch/chatDispatcher/groupFuncDispatcher are lowercase and read as one coherent style.

The one place it's genuinely inconsistent is items.go: AddItem/DropItem (capitalized) and itemTake/itemDrop (lowercase) are four methods on Player doing the same category of work — moving an item into or out of the inventory — but two are capitalized and two aren't, with no functional reason for the split (none of these are called outside package main, so capitalization here is purely stylistic, not enforced by the compiler). Worth picking one convention and applying it to all four.

Separately, NewPlayer, CleanupPlayerData, and the Group* family (GroupCreate/GroupInvite/GroupJoin/GroupLeave/GroupDisband) are all capitalized too — that's internally consistent with each other (constructors and the group lifecycle read as a related set), just capitalized compared to the mostly-lowercase handler style used everywhere else. Not wrong, just a second style cluster living alongside the first — your call whether that split (concept-level lifecycle/constructor funcs vs. everything else) is intentional or worth flattening.

One more, spanning player.go and items.go together: CleanupPlayerData (player.go) is a method on Player, capitalized just like AddItem/DropItem, but it sits alongside 8 other Player methods that are all lowercase (isOnTimeout, handleTimeoutBucket, itemTake, itemDrop, getZoneID, getPlayerName, getPlayerGroupInfo). So it's really one inconsistency across three methods (CleanupPlayerData, AddItem, DropItem) against eight, not two separate issues — worth fixing as one decision rather than two notes.



Left unchecked, if you want to close the gap:

TalkToAction.Target and BattleAction.Target aren't checked against the npcs map — a quest step can point at an NPC that doesn't exist
EnterAreaAction.Area isn't checked against zones
BattleAction.Target isn't checked for being attackable/enemy-role, so a battle step could target an NPC that can't be fought


if l := len(zones); l < 1 { // NEEDS TO BE 8
		errs = append(errs, fmt.Errorf("VALIDATION_ERROR: INVALID_ROOM_COUNT %d", l))
	}