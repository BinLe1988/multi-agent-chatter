package api

import (
	"net/http"

	"github.com/BinLe1988/multi-agent-chatter/pkg/filter"
	"github.com/BinLe1988/multi-agent-chatter/pkg/filter/model"

	"github.com/gin-gonic/gin"
)

// ContentFilterHandler 内容过滤处理器
type ContentFilterHandler struct {
	filterService *filter.ContentFilterService
}

// NewContentFilterHandler 创建新的内容过滤处理器
func NewContentFilterHandler() *ContentFilterHandler {
	return &ContentFilterHandler{
		filterService: filter.NewContentFilterService(filter.LevelMedium),
	}
}

// RegisterRoutes 注册路由
func (h *ContentFilterHandler) RegisterRoutes(router *gin.Engine) {
	filterGroup := router.Group("/api/filter")
	{
		filterGroup.POST("/check", h.CheckContent)
		filterGroup.POST("/config", h.UpdateConfig)
		filterGroup.POST("/sensitive-words", h.UpdateSensitiveWords)
		filterGroup.POST("/patterns", h.AddPattern)
	}
}

// CheckRequest 检查请求
type CheckRequest struct {
	Content     string            `json:"content" binding:"required"`
	ContentType model.ContentType `json:"content_type" binding:"required"`
}

// CheckContent 检查内容
func (h *ContentFilterHandler) CheckContent(c *gin.Context) {
	var req CheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.filterService.Filter(c.Request.Context(), req.Content, req.ContentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConfigRequest 配置请求
type ConfigRequest struct {
	Level filter.FilterLevel `json:"level" binding:"required"`
}

// UpdateConfig 更新配置
func (h *ContentFilterHandler) UpdateConfig(c *gin.Context) {
	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建新的过滤服务
	h.filterService = filter.NewContentFilterService(req.Level)

	c.JSON(http.StatusOK, gin.H{
		"message": "Filter configuration updated successfully",
		"level":   req.Level,
	})
}

// SensitiveWordsRequest 敏感词请求
type SensitiveWordsRequest struct {
	Words []string `json:"words" binding:"required"`
}

// UpdateSensitiveWords 更新敏感词列表
func (h *ContentFilterHandler) UpdateSensitiveWords(c *gin.Context) {
	var req SensitiveWordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.filterService.LoadSensitiveWords(req.Words)

	c.JSON(http.StatusOK, gin.H{
		"message": "Sensitive words updated successfully",
		"count":   len(req.Words),
	})
}

// PatternRequest 正则表达式模式请求
type PatternRequest struct {
	Pattern string `json:"pattern" binding:"required"`
}

// AddPattern 添加正则表达式模式
func (h *ContentFilterHandler) AddPattern(c *gin.Context) {
	var req PatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.filterService.AddRegexPattern(req.Pattern); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pattern added successfully",
		"pattern": req.Pattern,
	})
}
