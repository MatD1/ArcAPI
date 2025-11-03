package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

type Mission struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ExternalID    string    `gorm:"uniqueIndex;not null" json:"external_id"`
	Name          string    `gorm:"not null" json:"name"`
	Description   string    `gorm:"type:text" json:"description"`
	Trader        string    `json:"trader,omitempty"`
	Objectives    JSONB     `gorm:"type:jsonb" json:"objectives,omitempty"`      // Array of strings
	RewardItemIds JSONB     `gorm:"type:jsonb" json:"reward_item_ids,omitempty"` // Array of {itemId, quantity}
	XP            int       `json:"xp,omitempty"`
	Data          JSONB     `gorm:"type:jsonb" json:"data,omitempty"`
	SyncedAt      time.Time `json:"synced_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Mission) TableName() string {
	return "missions"
}
