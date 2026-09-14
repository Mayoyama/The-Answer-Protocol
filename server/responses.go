package main

// RoomInfo describes a zone for JSON responses.
type RoomInfo struct {
	RoomID      string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

// LookResponse is the JSON payload for the LOOK command.
type LookResponse struct {
	Room    RoomInfo `json:"room"`
	Players []string `json:"players"`
	Items   []string `json:"items"`
	NPCS    []string `json:"npcs"`
}

// PlayerStatusResponse is the JSON payload for the STATUS command.
type PlayerStatusResponse struct {
	HP     int    `json:"hp"`
	MaxHP  int    `json:"max_hp"`
	Status string `json:"status"`
}

// AttackResponse is the JSON payload for the ATTACK command.
type AttackResponse struct {
	AttackerHP int    `json:"attacker_hp"`
	TargetHP   int    `json:"target_hp"`
	Damage     int    `json:"damage"`
	Status     string `json:"status"`
}

// QuestResponse is the JSON payload for a single quest.
type QuestResponse struct {
	QuestID     string `json:"quest_id"`
	Description string `json:"description"`
	Reward      string `json:"reward"`
	Status      string `json:"status"`
}

// QuestSummary is the condensed quest info listed in QuestsResponse.
type QuestSummary struct {
	QuestID  string `json:"quest_id"`
	Status   string `json:"status"`
	Progress string `json:"progress,omitempty"`
}

// WhoResponse is the JSON payload for the WHO command.
type WhoResponse struct {
	RoomPlayers []string `json:"room"`
	ServerCount int      `json:"server"`
}

// QuestsResponse is the JSON payload for the QUESTS command.
type QuestsResponse []QuestSummary

// InventoryResponse is the JSON payload for the INVENTORY command.
type InventoryResponse []string
