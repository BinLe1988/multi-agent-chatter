package api

import (
	"net/http"
	"strconv"

	"github.com/BinLe1988/multi-agent-chatter/models"
	"github.com/BinLe1988/multi-agent-chatter/pkg/matching"

	"github.com/gin-gonic/gin"
)

type MatchingHandler struct {
	matcher *matching.Matcher
}

func NewMatchingHandler() *MatchingHandler {
	return &MatchingHandler{
		matcher: matching.NewMatcher(),
	}
}

// RegisterRoutes 注册路由
func (h *MatchingHandler) RegisterRoutes(router *gin.Engine) {
	matchGroup := router.Group("/api/match")
	{
		matchGroup.GET("/recommend/:userId", h.GetRecommendations)
		matchGroup.POST("/update-profile", h.UpdateUserProfile)
		matchGroup.GET("/profile/:userId", h.GetUserProfile)
	}
}

// GetRecommendations 获取推荐匹配
func (h *MatchingHandler) GetRecommendations(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// 获取用户画像
	var user models.UserProfile
	// TODO: 从数据库获取用户画像
	user.ID = uint(userID)

	// 获取候选用户列表
	var candidates []*models.UserProfile
	// TODO: 从数据库获取候选用户列表

	// 执行匹配
	matches := h.matcher.Match(&user, candidates)

	// 返回前N个最佳匹配
	const maxResults = 10
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	c.JSON(http.StatusOK, gin.H{
		"matches": matches,
	})
}

// UpdateUserProfile 更新用户画像
func (h *MatchingHandler) UpdateUserProfile(c *gin.Context) {
	var profile models.UserProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 保存到数据库

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"profile": profile,
	})
}

// GetUserProfile 获取用户画像
func (h *MatchingHandler) GetUserProfile(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// TODO: 从数据库获取用户画像
	var profile models.UserProfile
	profile.ID = uint(userID)

	c.JSON(http.StatusOK, gin.H{
		"profile": profile,
	})
}
