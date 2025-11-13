package models

import (
	"time"
)

type AuditLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	APIKeyID       *uint     `gorm:"index" json:"api_key_id,omitempty"`
	APIKey         *APIKey   `gorm:"foreignKey:APIKeyID" json:"api_key,omitempty"`
	JWTTokenID     *uint     `gorm:"index" json:"jwt_token_id,omitempty"`
	JWTToken       *JWTToken `gorm:"foreignKey:JWTTokenID" json:"jwt_token,omitempty"`
	UserID         *uint     `gorm:"index" json:"user_id,omitempty"`
	User           *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Endpoint       string    `gorm:"not null;index" json:"endpoint"`
	Method         string    `gorm:"not null;index" json:"method"`
	StatusCode     int       `gorm:"not null;index" json:"status_code"`
	RequestBody    *JSONB    `gorm:"type:jsonb" json:"request_body,omitempty"`
	ResponseTimeMs int64     `gorm:"not null" json:"response_time_ms"`
	IPAddress      string    `gorm:"index" json:"ip_address"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
