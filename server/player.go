package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// Status is a player's health/combat state.
type Status int

// Status values.
const (
	Healthy Status = iota
	Weakened
	Injured
	Engaged
	Unknown
)

// onlinePlayers holds all currently connected players, keyed by username.
var (
	onlinePlayers   = make(map[string]*Player)
	onlinePlayersMu sync.Mutex
)

// Player represents a connected player and their in-game state.
type Player struct {
	Username       string
	MaxHP          int
	CurrHP         int
	CurrLoc        string
	Status         Status
	BattleStats    PlayerBattleStats
	GroupInfo      *Group
	Quests         map[string]*PlayerQuest
	Inventory      map[string]bool
	KeyInventory   map[string]bool
	Gold           int
	Conn           net.Conn
	TokenCount     float64
	BucketTS       time.Time
	TimeoutEnd     time.Time
	TimeoutActions int
	PlayerMu       sync.Mutex
}

// PlayerBattleStats holds a player's combat stats.
type PlayerBattleStats struct {
	Strength    int
	BattleSkill int
	Dexterity   int
}

// newPlayer creates a new Player at the given starting location.
func newPlayer(username, startLoc string, conn net.Conn) *Player {
	battleStats := PlayerBattleStats{
		Strength:    7,
		BattleSkill: 7,
		Dexterity:   10,
	}
	return &Player{
		Username:     username,
		MaxHP:        100,
		CurrHP:       100,
		CurrLoc:      startLoc,
		Status:       Healthy,
		BattleStats:  battleStats,
		Inventory:    make(map[string]bool),
		KeyInventory: make(map[string]bool),
		Quests:       make(map[string]*PlayerQuest),
		Conn:         conn,
	}
}

// setPlayerHPStatus sets the player's status from their HP percentage; errors if MaxHP <= 0.
func (p *Player) setPlayerHPStatus() error {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	if p.MaxHP <= 0 {
		return fmt.Errorf("%w: PLAYER_MAXHP_0", InternalErr)
	}

	hpp := float64(p.CurrHP) / float64(p.MaxHP)

	switch {
	case hpp <= 0.3:
		p.Status = Injured
	case hpp <= 0.6:
		p.Status = Weakened
	default:
		p.Status = Healthy
	}

	return nil
}

// restoreSelfHP heals the player up to MaxHP and returns the amount actually healed.
func (p *Player) restoreSelfHP(amount int) int {
	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()

	var healedAmount int

	if amount < 0 {
		slog.Warn(InternalErr.Error(), "player", p.Username, "action", "restoreSelfHP", "amount", amount)
		amount = 0
	}

	switch {
	case p.CurrHP+amount > p.MaxHP:
		healedAmount = p.MaxHP - p.CurrHP
		p.CurrHP = p.MaxHP
	default:
		healedAmount = amount
		p.CurrHP += amount
	}

	return healedAmount
}

// cleanupPlayerData ends the player's fight, leaves their group, returns carried items,
// and removes them from their zone and the online list.
func (p *Player) cleanupPlayerData() {
	inPT := p.getPlayerGroupInfo()
	pname := p.getPlayerName()

	OngoingBattlesMu.Lock()
	battle, ok := OngoingBattles[pname]
	OngoingBattlesMu.Unlock()

	if ok {
		terminateBattle(pname, battle.NPC)
		close(battle.PlayerAttack)
	}

	if inPT != nil {
		_ = inPT.groupLeave(p)
	}

	currLoc := p.getZoneID()
	zone, zOK := getZoneObj(currLoc)

	if zOK {
		var crossZoneItems []*Item

		zone.ZoneMu.Lock()
		delete(zone.InZone, pname)

		for _, pl := range zone.InZone {
			_, _ = fmt.Fprintln(pl.Conn, EvtZoneLeave(pname))
			slog.Info("SYS_MESSAGE", "player", pl.getPlayerName(), "message", EvtZoneLeave(pname), "reason", "player cleanup")
		}

		for i, b := range p.Inventory {
			if b {
				item, ok := resolveItem(i)

				if !ok {
					continue
				}

				if item.BaseLoc == zone {
					zone.Items[i] = item

				} else {
					crossZoneItems = append(crossZoneItems, item)
				}
			}
		}

		zone.ZoneMu.Unlock()

		for _, item := range crossZoneItems {
			item.BaseLoc.ZoneMu.Lock()
			item.BaseLoc.Items[item.ItemID] = item
			item.BaseLoc.ZoneMu.Unlock()
		}
	}

	onlinePlayersMu.Lock()
	_, pOK := onlinePlayers[pname]

	if pOK {
		delete(onlinePlayers, pname)
	}

	for k, pl := range onlinePlayers {
		_, _ = fmt.Fprintln(pl.Conn, EvtPlayerCount(len(onlinePlayers)))
		slog.Info("SYS_MESSAGE", "recipient", k, "message", EvtPlayerCount(len(onlinePlayers)), "reason", "player left server")
	}

	onlinePlayersMu.Unlock()

	slog.Info("PLAYER_CLEANUP_COMPLETE", "player", pname)
}

// printStatus sends the player's HP/status as a JSON response.
func printStatus(player *Player, command, args string) {
	pStatus := getPlayerStatus(player)

	player.PlayerMu.Lock()

	playerStats := PlayerStatusResponse{
		HP:     player.CurrHP,
		MaxHP:  player.MaxHP,
		Status: pStatus,
	}

	statPrint, err := json.Marshal(playerStats)
	player.PlayerMu.Unlock()

	if err != nil {
		_, _ = fmt.Fprintln(player.Conn, InternalErr.Error())
		slog.Error(JSONErr.Error(), "player", player.getPlayerName(), "command", command, "args", args)

		return
	}

	_, _ = fmt.Fprintln(player.Conn, "OK", string(statPrint))
	slog.Info("SYS_MESSAGE", "player", player.getPlayerName(), "message", "OK "+string(statPrint), "command", command)
}

// printGoldBalance sends the player's gold as "OK <amount> GOLD".
func printGoldBalance(player *Player) {
	pname := player.getPlayerName()

	player.PlayerMu.Lock()
	msg := fmt.Sprintf("OK %d GOLD", player.Gold)
	player.PlayerMu.Unlock()

	_, _ = fmt.Fprintln(player.Conn, msg)
	slog.Info("SYS_MESSAGE", "player", pname, "message", msg, "command", "GOLD")

}
