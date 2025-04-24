package models

import (
	"gorm.io/gorm"
	"time"
)

// 支付状态
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentCompleted PaymentStatus = "completed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

// 支付方式
type PaymentMethod string

const (
	PaymentAlipay     PaymentMethod = "alipay"
	PaymentWechat     PaymentMethod = "wechat"
	PaymentUnionPay   PaymentMethod = "unionpay"
	PaymentCreditCard PaymentMethod = "creditcard"
)

// Payment 支付记录
type Payment struct {
	gorm.Model
	UserID        uint          `gorm:"not null" json:"userId"`
	OrderNo       string        `gorm:"size:50;not null;unique" json:"orderNo"`
	Amount        float64       `gorm:"not null" json:"amount"`
	Credits       int           `gorm:"not null" json:"credits"`
	Method        PaymentMethod `gorm:"size:20;not null" json:"method"`
	Status        PaymentStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	CompletedAt   *time.Time    `json:"completedAt"`
	TransactionID string        `gorm:"size:100" json:"transactionId"`
}

// RechargePackage 充值套餐
type RechargePackage struct {
	ID       string  `json:"id"`
	Credits  int     `json:"credits"`
	Price    float64 `json:"price"`
	Discount int     `json:"discount"`
}

// RechargeRequest 充值请求
type RechargeRequest struct {
	PackageID    string        `json:"packageId"`
	CustomAmount int           `json:"customAmount"`
	Method       PaymentMethod `json:"method" binding:"required"`
}

// PaymentResponse 支付响应
type PaymentResponse struct {
	OrderNo     string        `json:"orderNo"`
	Amount      float64       `json:"amount"`
	Credits     int           `json:"credits"`
	Method      PaymentMethod `json:"method"`
	Status      PaymentStatus `json:"status"`
	PaymentURL  string        `json:"paymentUrl,omitempty"`
	PaymentQR   string        `json:"paymentQr,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
}

// GetDefaultRechargePackages 获取默认套餐
func GetDefaultRechargePackages() []RechargePackage {
	return []RechargePackage{
		{ID: "small", Credits: 100, Price: 10, Discount: 0},
		{ID: "medium", Credits: 500, Price: 45, Discount: 10},
		{ID: "large", Credits: 1200, Price: 100, Discount: 15},
		{ID: "xlarge", Credits: 3000, Price: 230, Discount: 20},
	}
}
