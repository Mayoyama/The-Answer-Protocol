package main

type RoomInfo struct {
	RoomID      string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

type LookResponse struct {
	Room    RoomInfo `json:"room"`
	Players []string `json:"players"`
	Items   []string `json:"items"`
	NPCS    []string `json:"npcs"`
}

type PlayerStatusResponse struct {
	HP     int    `json:"hp"`
	MaxHP  int    `json:"max_hp"`
	Status string `json:"status"`
}

type AttackResponse struct {
	AttackerHP int    `json:"attacker_hp"`
	TargetHP   int    `json:"target_hp"`
	Damage     int    `json:"damage"`
	Status     string `json:"status"`
}

type QuestResponse struct {
	QuestID     string `json:"quest_id"`
	Description string `json:"description"`
	Reward      string `json:"reward"`
	Status      string `json:"status"`
}

type QuestSummary struct {
	QuestID  string `json:"quest_id"`
	Status   string `json:"status"`
	Progress string `json:"progress,omitempty"`
}

type QuestsResponse []QuestSummary

type InventoryResponse []string
