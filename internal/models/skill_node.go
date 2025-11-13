package models

import (
	"time"
)

type SkillNode struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	ExternalID          string    `gorm:"uniqueIndex;not null" json:"external_id"`
	Name                string    `gorm:"not null" json:"name"`
	Description         string    `gorm:"type:text" json:"description"`
	ImpactedSkill       string    `json:"impacted_skill,omitempty"`
	KnownValue          JSONB     `gorm:"type:jsonb" json:"known_value,omitempty"` // Array
	Category            string    `json:"category,omitempty"`
	MaxPoints           int       `json:"max_points,omitempty"` // Maximum level/points for this skill node (from GitHub maxPoints)
	IconName            string    `json:"icon_name,omitempty"`
	IsMajor             bool      `json:"is_major,omitempty"`
	Position            JSONB     `gorm:"type:jsonb" json:"position,omitempty"`              // {x, y}
	PrerequisiteNodeIds JSONB     `gorm:"type:jsonb" json:"prerequisite_node_ids,omitempty"` // Array of strings
	Data                JSONB     `gorm:"type:jsonb" json:"data,omitempty"`
	SyncedAt            time.Time `json:"synced_at"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (SkillNode) TableName() string {
	return "skill_nodes"
}
