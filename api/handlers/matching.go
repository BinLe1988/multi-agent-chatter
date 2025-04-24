package handlers

import (
	"net/http"
	"time"

	"github.com/BinLe1988/multi-agent-chatter/database"
	"github.com/BinLe1988/multi-agent-chatter/models"

	"github.com/gin-gonic/gin"
)

// 维护等待匹配的用户队列
var waitingUsers = make(map[uint]*models.MatchingRequest)

// RequestMatching 请求匹配
func RequestMatching(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req models.MatchingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 将用户添加到等待队列
	waitingUsers[userID.(uint)] = &req

	// 尝试匹配用户
	matched := false
	var matchedUserID uint

	for id := range waitingUsers {
		if id == userID.(uint) {
			continue // 跳过自己
		}

		// 简单匹配逻辑，可以根据兴趣等条件进行更复杂的匹配
		// 这里仅作为示例
		matched = true
		matchedUserID = id
		delete(waitingUsers, id) // 从等待队列中移除
		break
	}

	if matched {
		// 创建聊天会话
		session := models.ChatSession{
			UserID:     userID.(uint),
			Type:       models.SessionStranger,
			Title:      "陌生人聊天",
			LastActive: time.Now(),
			Meta:       `{"matchedUserId": ` + string(matchedUserID) + `}`,
		}

		if err := database.DB.Create(&session).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat session"})
			return
		}

		// 创建对方的会话
		matchedSession := models.ChatSession{
			UserID:     matchedUserID,
			Type:       models.SessionStranger,
			Title:      "陌生人聊天",
			LastActive: time.Now(),
			Meta:       `{"matchedUserId": ` + string(userID.(uint)) + `}`,
		}

		if err := database.DB.Create(&matchedSession).Error; err != nil {
			// 如果创建失败，删除之前创建的会话
			database.DB.Delete(&session)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create matched chat session"})
			return
		}

		// 发送系统消息
		welcomeMsg := "您已与一位陌生人匹配成功，开始聊天吧！"

		// 在两个会话中都添加系统消息
		systemMessage1 := models.ChatMessage{
			SessionID: session.ID,
			SenderID:  "system",
			Type:      models.MessageText,
			Content:   welcomeMsg,
		}

		systemMessage2 := models.ChatMessage{
			SessionID: matchedSession.ID,
			SenderID:  "system",
			Type:      models.MessageText,
			Content:   welcomeMsg,
		}

		database.DB.Create(&systemMessage1)
		database.DB.Create(&systemMessage2)

		// 从等待队列中移除自己
		delete(waitingUsers, userID.(uint))

		c.JSON(http.StatusOK, gin.H{
			"matched":   true,
			"sessionId": session.ID,
			"message":   "匹配成功，可以开始聊天了！",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"matched": false,
			"message": "已加入匹配队列，请耐心等待...",
		})
	}
}

// CancelMatching 取消匹配
func CancelMatching(c *gin.Context) {
	userID, _ := c.Get("userID")

	// 从等待队列中移除用户
	delete(waitingUsers, userID.(uint))

	c.JSON(http.StatusOK, gin.H{
		"message": "匹配已取消",
	})
}

// GetMatchingStatus 获取匹配状态
func GetMatchingStatus(c *gin.Context) {
	userID, _ := c.Get("userID")

	// 检查用户是否在等待队列中
	_, isWaiting := waitingUsers[userID.(uint)]

	if isWaiting {
		c.JSON(http.StatusOK, gin.H{
			"status":  "waiting",
			"message": "正在等待匹配...",
		})
		return
	}

	// 检查是否已经匹配成功（查找最近的陌生人聊天会话）
	var session models.ChatSession
	err := database.DB.Where("user_id = ? AND type = ? AND created_at > ?",
		userID, models.SessionStranger, time.Now().Add(-5*time.Minute)).
		Order("created_at DESC").
		First(&session).Error

	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "matched",
			"sessionId": session.ID,
			"message":   "匹配成功",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "idle",
		"message": "当前未在匹配中",
	})
}
