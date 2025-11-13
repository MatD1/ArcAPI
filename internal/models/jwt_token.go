package models

import (
	"time"
)

type JWTToken struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	User      User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TokenHash string     `gorm:"not null;index" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (JWTToken) TableName() string {
	return "jwt_tokens"
}

func (t *JWTToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t *JWTToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
