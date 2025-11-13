package models

import (
	"time"
)

// UserQuestProgress tracks which quests a user has completed
type UserQuestProgress struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_user_quest;not null" json:"user_id"`
	QuestID   uint      `gorm:"uniqueIndex:idx_user_quest;not null" json:"quest_id"`
	Completed bool      `gorm:"default:false;not null" json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Quest Quest `gorm:"foreignKey:QuestID" json:"quest,omitempty"`
}

func (UserQuestProgress) TableName() string {
	return "user_quest_progress"
}

// UserHideoutModuleProgress tracks hideout module progress for a user
type UserHideoutModuleProgress struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"uniqueIndex:idx_user_hideout_module;not null" json:"user_id"`
	HideoutModuleID uint      `gorm:"uniqueIndex:idx_user_hideout_module;not null" json:"hideout_module_id"`
	Unlocked        bool      `gorm:"default:false;not null" json:"unlocked"`
	Level           int       `gorm:"default:0;not null" json:"level"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	User          User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	HideoutModule HideoutModule `gorm:"foreignKey:HideoutModuleID" json:"hideout_module,omitempty"`
}

func (UserHideoutModuleProgress) TableName() string {
	return "user_hideout_module_progress"
}

// UserSkillNodeProgress tracks skill node progress for a user
type UserSkillNodeProgress struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"uniqueIndex:idx_user_skill_node;not null" json:"user_id"`
	SkillNodeID uint      `gorm:"uniqueIndex:idx_user_skill_node;not null" json:"skill_node_id"`
	Unlocked    bool      `gorm:"default:false;not null" json:"unlocked"`
	Level       int       `gorm:"default:0;not null" json:"level"` // Current level (0 if not unlocked)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SkillNode SkillNode `gorm:"foreignKey:SkillNodeID" json:"skill_node,omitempty"`
}

func (UserSkillNodeProgress) TableName() string {
	return "user_skill_node_progress"
}

// UserBlueprintProgress tracks blueprint consumption for a user
type UserBlueprintProgress struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"uniqueIndex:idx_user_blueprint;not null" json:"user_id"`
	ItemID       uint      `gorm:"uniqueIndex:idx_user_blueprint;not null" json:"item_id"`
	Consumed     bool      `gorm:"default:false;not null" json:"consumed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Item Item `gorm:"foreignKey:ItemID" json:"item,omitempty"`
}

func (UserBlueprintProgress) TableName() string {
	return "user_blueprint_progress"
}
