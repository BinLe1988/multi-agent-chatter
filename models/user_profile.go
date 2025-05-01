package models

import (
	"time"
)

// UserProfile 用户画像模型
type UserProfile struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"uniqueIndex"`
	Interests []Interest `json:"interests" gorm:"many2many:user_interests"`
	Tags      []Tag      `json:"tags" gorm:"many2many:user_tags"`

	// 行为数据
	LastActive   time.Time `json:"last_active"`
	LoginCount   int       `json:"login_count"`
	MessageCount int       `json:"message_count"`
	ActiveHours  []int     `json:"active_hours" gorm:"type:json"` // 活跃时间段

	// 偏好设置
	PreferredLanguages []string `json:"preferred_languages" gorm:"type:json"`
	AgeRange           string   `json:"age_range"`
	Gender             string   `json:"gender"`
	Location           string   `json:"location"`

	// 互动数据
	InteractionScore float64   `json:"interaction_score"` // 互动活跃度评分
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Interest 兴趣模型
type Interest struct {
	ID    uint    `json:"id" gorm:"primaryKey"`
	Name  string  `json:"name"`
	Score float64 `json:"score"` // 兴趣强度评分
}

// Tag 标签模型
type Tag struct {
	ID     uint    `json:"id" gorm:"primaryKey"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`   // 标签类型：hobby/skill/personality等
	Weight float64 `json:"weight"` // 标签权重
}

// UserBehavior 用户行为记录
type UserBehavior struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id"`
	Type      string    `json:"type"`     // 行为类型：chat/login/browse等
	Target    string    `json:"target"`   // 行为对象
	Duration  int       `json:"duration"` // 行为持续时间（秒）
	CreatedAt time.Time `json:"created_at"`
}
