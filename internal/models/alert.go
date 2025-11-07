package models

import (
	"time"
)

type Alert struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Severity    string    `gorm:"not null" json:"severity"`         // e.g., "info", "warning", "error", "critical"
	IsActive    bool      `gorm:"default:true" json:"is_active"`    // Whether the alert is currently active
	Data        JSONB     `gorm:"type:jsonb" json:"data,omitempty"` // Full data including multilingual content
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Alert) TableName() string {
	return "alerts"
}
