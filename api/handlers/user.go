package handlers

import (
	"multi-agent-chatter/database"
	"multi-agent-chatter/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateSubscription 更新用户订阅
func UpdateSubscription(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req models.SubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取订阅计划
	plans := models.GetSubscriptionPlans()
	plan, exists := plans[string(req.Type)]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription type"})
		return
	}

	// 获取用户
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// 设置订阅过期时间
	expiresAt := time.Now().AddDate(0, 1, 0) // 1个月后

	// 更新用户订阅
	user.SubType = req.Type
	user.SubExpiresAt = &expiresAt
	user.SubAutoRenew = true

	// 增加积分
	user.Credits += plan.CreditsPerMonth

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscription updated successfully",
		"user":    user.ToResponse(),
	})
}

// GetSubscriptionPlans 获取订阅计划
func GetSubscriptionPlans(c *gin.Context) {
	plans := models.GetSubscriptionPlans()
	c.JSON(http.StatusOK, gin.H{
		"plans": plans,
	})
}

// UpdateUserProfile 更新用户资料
func UpdateUserProfile(c *gin.Context) {
	userID, _ := c.Get("userID")

	var update struct {
		Username string `json:"username"`
	}

	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	if update.Username != "" {
		var count int64
		database.DB.Model(&models.User{}).Where("username = ? AND id != ?", update.Username, userID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
	}

	// 更新用户资料
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	if update.Username != "" {
		user.Username = update.Username
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    user.ToResponse(),
	})
}
