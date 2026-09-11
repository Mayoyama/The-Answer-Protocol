package main

func getZoneObj(loc string) (*Zone, bool) {
	zonesMu.Lock()
	zone, ok := zones[loc]
	zonesMu.Unlock()

	return zone, ok
}

func (z *Zone) getZoneName() string {
	z.ZoneMu.Lock()
	zname := z.ZoneName
	z.ZoneMu.Unlock()

	return zname
}

func (p *Player) getZoneID() string {
	p.PlayerMu.Lock()
	loc := p.CurrLoc
	p.PlayerMu.Unlock()

	return loc
}

func (p *Player) getPlayerName() string {
	p.PlayerMu.Lock()
	name := p.Username
	p.PlayerMu.Unlock()

	return name
}

func (p *Player) getPlayerGroupInfo() *Group {
	p.PlayerMu.Lock()
	inPT := p.GroupInfo
	p.PlayerMu.Unlock()

	return inPT
}

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

func (n *NPC) getNPCName() string {
	n.NPCMu.Lock()
	sname := n.NPCName
	n.NPCMu.Unlock()

	return sname
}
