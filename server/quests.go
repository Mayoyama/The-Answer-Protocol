package main

import ()

// QuestType categorizes a quest's objective (delivery, fetch, or battle).
type QuestType int

const (
	Delivery QuestType = iota
	Fetch
	Battle
)

// Quest represents an in-game quest.
type Quest struct {
	Name        string
	QuestID     string
	Description string
	Type        QuestType
	Steps       []QuestAction
	Reward      *Reward
}

// Reward is a quest's payout: an optional key item and/or gold.
type Reward struct {
	KeyItem string `yaml:"key_item"`
	Money   int    `yaml:"gold"`
}

// QuestAction is one step in a quest's Steps sequence.
type QuestAction interface {
	ActionType() string
}

// EnterAreaAction is a quest step completed by entering the named zone.
type EnterAreaAction struct {
	Area string `yaml:"enter_area"`
}

// ActionType returns the step's discriminator key, "enter_area".
func (eaa *EnterAreaAction) ActionType() string { return "enter_area" }

// TalkToAction is a quest step completed by talking to the named NPC.
type TalkToAction struct {
	Target   string `yaml:"talk_to"`
	Dialogue string `yaml:"dialogue"`
}

// ActionType returns the step's discriminator key, "talk_to".
func (tta *TalkToAction) ActionType() string { return "talk_to" }

// BattleAction is a quest step completed by defeating the named NPC.
type BattleAction struct {
	Target string `yaml:"battle"`
}

// ActionType returns the step's discriminator key, "battle".
func (ba *BattleAction) ActionType() string { return "battle" }
