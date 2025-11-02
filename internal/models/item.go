package models

import (
	"time"
)

type Item struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ExternalID  string    `gorm:"uniqueIndex;not null" json:"external_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	ImageURL    string    `json:"image_url,omitempty"`
	Data        JSONB     `gorm:"type:jsonb" json:"data,omitempty"`
	SyncedAt    time.Time `json:"synced_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Item) TableName() string {
	return "items"
}
