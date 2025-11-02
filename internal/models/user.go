package models

import (
	"time"
)

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GithubID  *string   `gorm:"uniqueIndex;null" json:"github_id,omitempty"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Role      UserRole  `gorm:"type:varchar(20);default:'user';not null" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
