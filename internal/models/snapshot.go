package models

import "time"

// Snapshot represents a complete point-in-time state of all static collections.
// This is used by the frontend to hydrate its local SQLite database.
type Snapshot struct {
	Version        string           `json:"version"`
	SyncedAt       time.Time        `json:"synced_at"`
	Quests         []Quest          `json:"quests"`
	Items          []Item           `json:"items"`
	SkillNodes     []SkillNode      `json:"skill_nodes"`
	HideoutModules []HideoutModule  `json:"hideout_modules"`
	EnemyTypes     []EnemyType      `json:"enemy_types"`
	Alerts         []Alert          `json:"alerts"`
	Bots           []Bot            `json:"bots"`
	Maps           []Map            `json:"maps"`
	Traders        []Trader         `json:"traders"`
	Projects       []Project        `json:"projects"`
}
