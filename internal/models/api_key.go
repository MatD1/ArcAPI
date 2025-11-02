package models

import (
	"time"
)

type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	KeyHash    string     `gorm:"not null;uniqueIndex" json:"-"`
	Name       string     `gorm:"not null" json:"name"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (APIKey) TableName() string {
	return "api_keys"
}

func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}
