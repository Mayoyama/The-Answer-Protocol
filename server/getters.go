package main

import "fmt"

// getZoneObj looks up a zone by its ID.
func getZoneObj(loc string) (*Zone, bool) {
	zonesMu.Lock()
	zone, ok := zones[loc]
	zonesMu.Unlock()

	return zone, ok
}

// getZoneID returns the player's current zone ID.
func (p *Player) getZoneID() string {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	return p.CurrLoc
}

// getPlayerName returns the player's username.
func (p *Player) getPlayerName() string {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	return p.Username
}

// getPlayerHP returns the player's current HP.
func (p *Player) getPlayerHP() int {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	return p.CurrHP
}

// getPlayerMaxHP returns the player's max HP.
func (p *Player) getPlayerMaxHP() int {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	return p.MaxHP
}

// getPlayerStatus returns the player's status as a lowercase string for responses.
func getPlayerStatus(player *Player) string {
	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	switch player.Status {
	case Healthy:
		return "healthy"
	case Weakened:
		return "weakened"
	case Injured:
		return "injured"
	case Engaged:
		return "engaged"
	default:
		return "unknown"
	}
}

// getPlayerGroupInfo returns the group the player belongs to, or nil.
func (p *Player) getPlayerGroupInfo() *Group {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	return p.GroupInfo
}

// getNPCObject resolves an NPC by name and confirms it's present in this zone.
func (z *Zone) getNPCObject(name string) (*NPC, bool) {
	n, ok := resolveNPC(name)
	if !ok {
		return nil, false
	}

	z.ZoneMu.Lock()
	_, exists := z.NPCs[n.NPCID]
	z.ZoneMu.Unlock()

	if !exists {
		return nil, false
	}

	return n, true
}

// getNPCName returns the NPC's display name.
func (n *NPC) getNPCName() string {
	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()

	return n.NPCName
}

// getNPCRole returns the NPC's role as a string for responses and logs.
func (n *NPC) getNPCRole() string {
	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()
	switch n.Role {
	case General:
		return "general"
	case QuestGiver:
		return "questGiver"
	case Healer:
		return "healer"
	case Enemy:
		return "enemy"
	default:
		return "unknownNPCRole"
	}
}

// getNPCHealerString returns the healer's heal line with the "(You feel a warm glow)" suffix.
func (n *NPC) getNPCHealerString() string {
	const warmGlowString = "You feel a warm glow"
	n.NPCMu.Lock()
	baseHealString := n.HealDialogue
	n.NPCMu.Unlock()

	return fmt.Sprintf("%s (%s)", baseHealString, warmGlowString)
}

// getNPCStats returns a copy of the NPC's stats.
func (n *NPC) getNPCStats() NPCStats {
	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()

	return n.Stats
}

// getNPCQuest returns the first quest this NPC can currently offer the player — skipping quests
// the player already has and ones whose prerequisite isn't completed — or false if none are eligible.
func (n *NPC) getNPCQuest(player *Player) (string, *Quest, bool) {
	n.NPCMu.Lock()
	defer n.NPCMu.Unlock()

	if n.Quests == nil {
		return "", nil, false
	}

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	for k, q := range n.Quests {
		if _, alreadyHave := player.Quests[k]; alreadyHave {
			continue
		}

		if q.Requires != "" {
			prereq, ok := player.Quests[q.Requires]

			if !ok || prereq.Status != Completed {
				continue
			}
		}

		return k, q, true
	}

	return "", nil, false
}
