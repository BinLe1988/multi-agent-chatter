package models

import (
	"gorm.io/gorm"
	"time"
)

// 消息类型
type MessageType string

const (
	MessageText  MessageType = "text"
	MessageImage MessageType = "image"
	MessageVoice MessageType = "voice"
)

// 会话类型
type SessionType string

const (
	SessionAI       SessionType = "ai"
	SessionStranger SessionType = "stranger"
	SessionGroup    SessionType = "group"
)

// ChatSession 聊天会话
type ChatSession struct {
	gorm.Model
	UserID    uint        `gorm:"not null" json:"userId"`
	Type      SessionType `gorm:"size:20;not null" json:"type"`
	Title     string      `gorm:"size:100" json:"title"`
	LastActive time.Time   `json:"lastActive"`
	Meta      string      `gorm:"type:json" json:"meta"` // 存储元数据，比如AI会话的模型、参数等
	Messages  []ChatMessage `gorm:"foreignKey:SessionID" json:"-"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	gorm.Model
	SessionID uint        `gorm:"not null" json:"sessionId"`
	UserID    uint        `json:"userId"`
	SenderID  string      `gorm:"size:50;not null" json:"senderId"` // 发送者ID，可能是用户ID或AI标识符
	Type      MessageType `gorm:"size:20;not null;default:'text'" json:"type"`
	Content   string      `gorm:"type:text" json:"content"`
	Metadata  string      `gorm:"type:json" json:"metadata"` // 存储额外的消息元数据
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID uint        `json:"sessionId"`
	Message   string      `json:"message" binding:"required"`
	Type      MessageType `json:"type" default:"text"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID        uint      `json:"id"`
	SessionID uint      `json:"sessionId"`
	SenderID  string    `json:"senderId"`
	Content   string    `json:"content"`
	Type      MessageType `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// MatchingRequest 匹配请求
type MatchingRequest struct {
	Interests []string `json:"interests"`
	AgeRange  [2]int   `json:"ageRange"`
	Gender    string   `json:"gender"`
}
