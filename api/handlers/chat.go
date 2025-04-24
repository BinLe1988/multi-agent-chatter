package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/BinLe1988/multi-agent-chatter/database"
	"github.com/BinLe1988/multi-agent-chatter/models"
	"github.com/BinLe1988/multi-agent-chatter/pkg/ai"

	"github.com/gin-gonic/gin"
)

// CreateChatSession 创建聊天会话
func CreateChatSession(c *gin.Context) {
	userID, _ := c.Get("userID")

	var session models.ChatSession
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session.UserID = userID.(uint)
	session.LastActive = time.Now()

	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"sessionId": session.ID,
		"message":   "Chat session created successfully",
	})
}

// GetChatSessions 获取所有聊天会话
func GetChatSessions(c *gin.Context) {
	userID, _ := c.Get("userID")

	var sessions []models.ChatSession
	if err := database.DB.Where("user_id = ?", userID).Order("last_active DESC").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
	})
}

// GetChatMessages 获取聊天消息
func GetChatMessages(c *gin.Context) {
	userID, _ := c.Get("userID")
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	// 验证会话所有权
	var session models.ChatSession
	if err := database.DB.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found or unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var messages []models.ChatMessage
	if err := database.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat messages"})
		return
	}

	// 获取消息总数
	var count int64
	database.DB.Model(&models.ChatMessage{}).Where("session_id = ?", sessionID).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"total":    count,
		"messages": messages,
	})
}

// SendChatMessage 发送聊天消息
func SendChatMessage(c *gin.Context) {
	userID, _ := c.Get("userID")
	user, _ := c.Get("user")

	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证会话所有权
	var session models.ChatSession
	if err := database.DB.Where("id = ? AND user_id = ?", req.SessionID, userID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found or unauthorized"})
		return
	}

	// 扣除积分（仅AI聊天）
	if session.Type == models.SessionAI {
		// 检查积分是否足够
		currentUser := user.(models.User)
		if currentUser.Credits < 1 {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient credits"})
			return
		}

		// 扣除积分
		currentUser.Credits--
		if err := database.DB.Save(&currentUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update credits"})
			return
		}
	}

	// 保存用户消息
	userMessage := models.ChatMessage{
		SessionID: req.SessionID,
		UserID:    userID.(uint),
		SenderID:  strconv.Itoa(int(userID.(uint))),
		Type:      req.Type,
		Content:   req.Message,
	}

	if err := database.DB.Create(&userMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send chat message"})
		return
	}

	// 如果是AI聊天，需要获取AI回复
	if session.Type == models.SessionAI {
		// 更新会话最后活动时间
		session.LastActive = time.Now()
		database.DB.Save(&session)

		// 调用AI服务获取回复
		aiResponse, err := ai.GenerateResponse(req.Message, session.Meta)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate AI response"})
			return
		}

		// 保存AI回复
		aiMessage := models.ChatMessage{
			SessionID: req.SessionID,
			UserID:    0, // AI消息没有用户ID
			SenderID:  "ai",
			Type:      models.MessageText,
			Content:   aiResponse,
		}

		if err := database.DB.Create(&aiMessage).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save AI response"})
			return
		}

		// 返回AI回复和用户消息
		c.JSON(http.StatusOK, gin.H{
			"message": userMessage,
			"reply":   aiMessage,
		})
		return
	}

	// 如果是陌生人匹配聊天，通知对方
	if session.Type == models.SessionStranger {
		// 这里应该通过WebSocket通知对方有新消息
		// TODO: 实现WebSocket通知逻辑

		c.JSON(http.StatusOK, gin.H{
			"message": userMessage,
		})
		return
	}

	// 普通消息直接返回
	c.JSON(http.StatusOK, gin.H{
		"message": userMessage,
	})
}
