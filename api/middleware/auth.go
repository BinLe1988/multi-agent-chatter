package middleware

import (
	"net/http"
	"strings"

	"github.com/BinLe1988/multi-agent-chatter/database"
	"github.com/BinLe1988/multi-agent-chatter/models"
	"github.com/BinLe1988/multi-agent-chatter/pkg/utils"

	"github.com/gin-gonic/gin"
)

// Auth 验证JWT令牌中间件
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// 将用户ID和用户信息存储在上下文中
		c.Set("userID", claims.UserID)
		c.Set("user", user)

		c.Next()
	}
}
