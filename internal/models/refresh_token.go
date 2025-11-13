package models

import "time"

// RefreshToken represents a long-lived token allowing JWT renewal
// Tokens are hashed at rest and can be revoked.
type RefreshToken struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	User       User       `gorm:"foreignKey:UserID" json:"-"`
	TokenHash  string     `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (t *RefreshToken) IsExpired() bool { return time.Now().After(t.ExpiresAt) }
func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }
