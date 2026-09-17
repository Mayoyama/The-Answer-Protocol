package main

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
	loc := p.CurrLoc
	p.PlayerMu.Unlock()

	return loc
}

// getPlayerName returns the player's username.
func (p *Player) getPlayerName() string {
	p.PlayerMu.Lock()
	name := p.Username
	p.PlayerMu.Unlock()

	return name
}

// getPlayerGroupInfo returns the group the player belongs to, or nil.
func (p *Player) getPlayerGroupInfo() *Group {
	p.PlayerMu.Lock()
	inPT := p.GroupInfo
	p.PlayerMu.Unlock()

	return inPT
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
	sname := n.NPCName
	n.NPCMu.Unlock()

	return sname
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
