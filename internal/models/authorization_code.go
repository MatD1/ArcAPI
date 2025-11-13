package models

import "time"

// AuthorizationCode represents a short-lived code used in PKCE auth flow
// Codes are one-time use and expire quickly (e.g. 60 seconds)
type AuthorizationCode struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"not null;index" json:"user_id"`
	User             User      `gorm:"foreignKey:UserID" json:"-"`
	CodeHash         string    `gorm:"not null;uniqueIndex" json:"-"`
	CodeChallenge    string    `gorm:"not null" json:"-"`
	CodeChallengeMethod string `gorm:"not null" json:"-"`
	ExpiresAt        time.Time `gorm:"not null;index" json:"expires_at"`
	ConsumedAt       *time.Time `json:"consumed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (AuthorizationCode) TableName() string { return "authorization_codes" }

func (c *AuthorizationCode) IsExpired() bool { return time.Now().After(c.ExpiresAt) }
func (c *AuthorizationCode) IsConsumed() bool { return c.ConsumedAt != nil }
