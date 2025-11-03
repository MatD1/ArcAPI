package models

import (
	"time"
)

type HideoutModule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ExternalID  string    `gorm:"uniqueIndex;not null" json:"external_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	MaxLevel    int       `json:"max_level,omitempty"`
	Levels      JSONB     `gorm:"type:jsonb" json:"levels,omitempty"` // Array of level objects
	Data        JSONB     `gorm:"type:jsonb" json:"data,omitempty"`
	SyncedAt    time.Time `json:"synced_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (HideoutModule) TableName() string {
	return "hideout_modules"
}
