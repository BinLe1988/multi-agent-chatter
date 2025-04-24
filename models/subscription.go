package models

// SubscriptionPlan 订阅计划
type SubscriptionPlan struct {
    Type           SubscriptionType `json:"type"`
    Name           string           `json:"name"`
    Price          float64          `json:"price"`
    CreditsPerMonth int              `json:"creditsPerMonth"`
    Features       []string         `json:"features"`
}

// SubscriptionRequest 订阅请求
type SubscriptionRequest struct {
    Type SubscriptionType `json:"type" binding:"required"`
}

// GetSubscriptionPlans 获取所有订阅计划
func GetSubscriptionPlans() map[string]SubscriptionPlan {
    return map[string]SubscriptionPlan{
        "free": {
            Type:           SubscriptionFree,
            Name:           "免费版",
            Price:          0,
            CreditsPerMonth: 100,
            Features:       []string{"基础聊天功能", "随机匹配", "基本AI助手"},
        },
        "basic": {
            Type:           SubscriptionBasic,
            Name:           "基础版",
            Price:          19.9,
            CreditsPerMonth: 500,
            Features:       []string{"所有免费功能", "专业AI助手", "无广告体验", "优先匹配"},
        },
        "premium": {
            Type:           SubscriptionPremium,
            Name:           "高级版",
            Price:          39.9,
            CreditsPerMonth: 1200,
            Features:       []string{"所有基础功能", "专属聊天定制", "创建聊天室", "语音转文字"},
        },
        "unlimited": {
            Type:           SubscriptionUnlimited,
            Name:           "无限版",
            Price:          99.9,
            CreditsPerMonth: 3000,
            Features:       []string{"所有高级功能", "无限AI助手使用", "VIP客户支持", "专属定制服务"},
        },
    }
}
