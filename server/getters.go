package main

func getZoneObj(loc string) (*Zone, bool) {
	zonesMu.Lock()
	zone, ok := zones[loc]
	zonesMu.Unlock()

	return zone, ok
}

func (p *Player) getZoneID() string {
	p.PlayerMu.Lock()
	loc := p.CurrLoc
	p.PlayerMu.Unlock()

	return loc
}

func (p *Player) getPlayerName() string {
	p.PlayerMu.Lock()
	name := p.getPlayerName()
	p.PlayerMu.Unlock()

	return name
}

func (p *Player) getPlayerGroupInfo() *Group {
	p.PlayerMu.Lock()
	inPT := p.GroupInfo
	p.PlayerMu.Unlock()

	return inPT
}
