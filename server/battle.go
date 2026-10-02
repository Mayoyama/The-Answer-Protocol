package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

var (
	OngoingBattles   = make(map[string]*BattleField)
	OngoingBattlesMu sync.Mutex
)

type BattleField struct {
	Player            *Player
	NPC               *NPC
	NPCCurrHP         int
	NPCBattleStats    NPCStats
	PlayerAttack      chan struct{}
	BattleRoundResult chan RoundResult
}

type BattleRound struct {
	DamageOutput   int
	InitiativeRoll int
	DodgedAttack   bool
}

type RoundResult struct {
	PlayerRound    BattleRound
	NPCRound       BattleRound
	PlayerDmgDealt int
	NPCDmgDealt    int
	PlayerHP       int
	NPCHP          int
}

func resolveAttackRequest(target string, player *Player) error {
	var battle *BattleField
	pname := player.getPlayerName()
	player.PlayerMu.Lock()
	pMaxHP := player.MaxHP
	player.PlayerMu.Unlock()

	if getPlayerStatus(player) == "engaged" {
		OngoingBattlesMu.Lock()
		fight, ok := OngoingBattles[pname]
		OngoingBattlesMu.Unlock()

		if !ok {
			return InternalErr
		}
		battle = fight

	} else {

		currLoc := player.getZoneID()
		currZone, ok := getZoneObj(currLoc)

		if !ok {
			return InternalErr
		}

		enemy, ok := currZone.getNPCObject(target)

		if !ok {
			return NPCNotFoundErr
		}

		err := enemy.canBeFought()

		if err != nil {
			return err
		}

		npcStats := enemy.getNPCStats()

		bf := &BattleField{
			Player:            player,
			NPC:               enemy,
			NPCCurrHP:         npcStats.HP,
			NPCBattleStats:    npcStats,
			PlayerAttack:      make(chan struct{}),
			BattleRoundResult: make(chan RoundResult),
		}

		OngoingBattlesMu.Lock()
		OngoingBattles[pname] = bf
		OngoingBattlesMu.Unlock()

		player.PlayerMu.Lock()
		player.Status = Engaged
		player.PlayerMu.Unlock()

		battle = bf

		go bf.processBattle()
	}

	battle.PlayerAttack <- struct{}{}
	result := <-battle.BattleRoundResult

	atkResponse := AttackResponse{
		AttackerInitRoll: result.PlayerRound.InitiativeRoll,
		TargetInitRoll:   result.NPCRound.InitiativeRoll,
		AttackerHP:       result.PlayerHP,
		AttackerMaxHP:    pMaxHP,
		TargetHP:         result.NPCHP,
		TargetMaxHP:      battle.NPC.getNPCStats().HP,
		TargetDodged:     result.NPCRound.DodgedAttack,
		Damage:           result.PlayerDmgDealt,
		AttackerDodged:   result.PlayerRound.DodgedAttack,
		DamageReceived:   result.NPCDmgDealt,
		Status:           getPlayerStatus(player),
	}

	attackResponse, err := json.Marshal(atkResponse)

	if err != nil {
		return JSONErr
	}

	_, _ = fmt.Fprintf(player.Conn, "OK %s\n", string(attackResponse))
	slog.Info("SYS_MESSAGE", "player", pname, "message", string(attackResponse), "command", "ATTACK", "npc", battle.NPC.getNPCName())

	if result.PlayerHP <= 0 {
		_, err := respawnPlayer(player)
		return err
	}

	return nil
}

func (enemy *NPC) canBeFought() error {

	enemy.NPCMu.Lock()
	defer enemy.NPCMu.Unlock()
	if enemy.Attackable {
		if enemy.Role == Enemy && !enemy.Occupied {
			enemy.Occupied = true
			return nil
		}

		return NPCOccupiedErr
	}

	return NPCNotHostileErr
}

func calcBattleRound(enemy *NPC, player *Player) (pBattleRound, npcBattleRound BattleRound) {
	player.PlayerMu.Lock()
	pDmgOutput := player.BattleStats.Strength + generateRandInt(1, player.BattleStats.BattleSkill)
	pDex := player.BattleStats.Dexterity
	player.PlayerMu.Unlock()

	pInitRoll := generateRandInt(1, 6)
	var pDodgedAttack bool

	if generateRandInt(1, 100) > pDex {
		pDodgedAttack = false

	} else {
		pDodgedAttack = true
	}

	pBattleRound = BattleRound{
		DamageOutput:   pDmgOutput,
		InitiativeRoll: pInitRoll,
		DodgedAttack:   pDodgedAttack,
	}

	enemy.NPCMu.Lock()
	npcDmgOutput := enemy.Stats.Strength + generateRandInt(1, enemy.Stats.BattleSkill)
	npcDex := enemy.Stats.Dexterity
	enemy.NPCMu.Unlock()

	npcInitRoll := generateRandInt(1, 6)
	var npcDodgedAttack bool

	if generateRandInt(1, 100) > npcDex {
		npcDodgedAttack = false

	} else {
		npcDodgedAttack = true
	}

	npcBattleRound = BattleRound{
		DamageOutput:   npcDmgOutput,
		InitiativeRoll: npcInitRoll,
		DodgedAttack:   npcDodgedAttack,
	}

	return pBattleRound, npcBattleRound
}

func calcDmgDealt(damageOutput int, attackDodged bool) int {
	if attackDodged {
		return 0
	}

	return damageOutput
}

func (bf *BattleField) applyDmgToPlayer(damage int) (int, bool) {
	bf.Player.PlayerMu.Lock()
	defer bf.Player.PlayerMu.Unlock()

	bf.Player.CurrHP -= damage

	return bf.Player.CurrHP, bf.Player.CurrHP <= 0
}

func (bf *BattleField) processBattle() {
	pname := bf.Player.getPlayerName()

	for {
		<-bf.PlayerAttack // wait until the "ATTACK" command

		var (
			pDamageDealt   int
			npcDamageDealt int
			battleOver     bool
			playerHP       int
		)

		pRound, enemyRound := calcBattleRound(bf.NPC, bf.Player)

		// player attacks first
		if pRound.InitiativeRoll >= enemyRound.InitiativeRoll {
			pDamageDealt = calcDmgDealt(pRound.DamageOutput, enemyRound.DodgedAttack)
			bf.NPCCurrHP -= pDamageDealt

			if bf.NPCCurrHP <= 0 {
				bf.NPCCurrHP = 0
				battleOver = true

				bf.Player.PlayerMu.Lock()
				playerHP = bf.Player.CurrHP
				bf.Player.PlayerMu.Unlock()

				pRound.DodgedAttack = false

			} else {
				npcDamageDealt = calcDmgDealt(enemyRound.DamageOutput, pRound.DodgedAttack)
				playerHP, battleOver = bf.applyDmgToPlayer(npcDamageDealt)
			}

			// NPC attacks first
		} else {
			npcDamageDealt = calcDmgDealt(enemyRound.DamageOutput, pRound.DodgedAttack)
			playerHP, battleOver = bf.applyDmgToPlayer(npcDamageDealt)

			if !battleOver {
				pDamageDealt = calcDmgDealt(pRound.DamageOutput, enemyRound.DodgedAttack)
				bf.NPCCurrHP -= pDamageDealt

				if bf.NPCCurrHP <= 0 {
					bf.NPCCurrHP = 0
					battleOver = true
				}

			} else {
				enemyRound.DodgedAttack = false
			}
		}

		if playerHP < 0 {
			playerHP = 0
		}

		roundRes := RoundResult{
			PlayerRound:    pRound,
			NPCRound:       enemyRound,
			PlayerDmgDealt: pDamageDealt,
			NPCDmgDealt:    npcDamageDealt,
			PlayerHP:       playerHP,
			NPCHP:          bf.NPCCurrHP,
		}

		if battleOver {
			bf.NPC.NPCMu.Lock()
			bf.NPC.Occupied = false
			bf.NPC.NPCMu.Unlock()

			OngoingBattlesMu.Lock()
			delete(OngoingBattles, pname)
			OngoingBattlesMu.Unlock()

			bf.Player.PlayerMu.Lock()
			if bf.Player.CurrHP <= 0 {
				bf.Player.CurrHP = bf.Player.MaxHP / 2
			}

			bf.Player.PlayerMu.Unlock()

			bf.Player.setPlayerHPStatus()

			bf.BattleRoundResult <- roundRes

			return
		}

		bf.BattleRoundResult <- roundRes
	}
}
