package models

import (
	"time"
)

type EnemyType struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ExternalID    string    `gorm:"uniqueIndex;not null" json:"external_id"`
	Name          string    `gorm:"not null" json:"name"`
	Description   string    `gorm:"type:text" json:"description,omitempty"`
	Type          string    `json:"type,omitempty"` // e.g., "Human", "Robot", "Alien"
	ImageURL      string    `json:"image_url,omitempty"`
	ImageFilename string    `json:"image_filename,omitempty"`
	Weakpoints    JSONB     `gorm:"type:jsonb" json:"weakpoints,omitempty"` // Array of weakpoint objects
	Data          JSONB     `gorm:"type:jsonb" json:"data,omitempty"`       // Full data including multilingual content
	SyncedAt      time.Time `json:"synced_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (EnemyType) TableName() string {
	return "enemy_types"
}
