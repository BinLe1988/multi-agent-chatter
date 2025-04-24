package models

import (
	"gorm.io/gorm"
	"time"
)

// 订阅类型
type SubscriptionType string

const (
	SubscriptionFree      SubscriptionType = "free"
	SubscriptionBasic     SubscriptionType = "basic"
	SubscriptionPremium   SubscriptionType = "premium"
	SubscriptionUnlimited SubscriptionType = "unlimited"
)

// User 用户模型
type User struct {
	gorm.Model
	Username     string          `gorm:"size:50;not null;unique" json:"username"`
	Email        string          `gorm:"size:100;not null;unique" json:"email"`
	Password     string          `gorm:"size:255;not null" json:"-"`
	Credits      int             `gorm:"default:100" json:"credits"`
	SubType      SubscriptionType `gorm:"size:20;default:'free'" json:"subType"`
	SubExpiresAt *time.Time      `json:"subExpiresAt"`
	SubAutoRenew bool            `gorm:"default:false" json:"subAutoRenew"`
	JoinDate     time.Time       `json:"joinDate"`
	LastLogin    *time.Time      `json:"lastLogin"`
}

// CredentialRequest 用户登录请求
type CredentialRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegistrationRequest 用户注册请求
type RegistrationRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID           uint            `json:"id"`
	Username     string          `json:"username"`
	Email        string          `json:"email"`
	Credits      int             `json:"credits"`
	Subscription struct {
		Type      SubscriptionType `json:"type"`
		ExpiresAt *time.Time      `json:"expiresAt"`
		AutoRenew bool            `json:"autoRenew"`
	} `json:"subscription"`
	JoinDate time.Time `json:"joinDate"`
}

// ToResponse 转换为响应
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Credits:  u.Credits,
		Subscription: struct {
			Type      SubscriptionType `json:"type"`
			ExpiresAt *time.Time      `json:"expiresAt"`
			AutoRenew bool            `json:"autoRenew"`
		}{
			Type:      u.SubType,
			ExpiresAt: u.SubExpiresAt,
			AutoRenew: u.SubAutoRenew,
		},
		JoinDate: u.JoinDate,
	}
}
