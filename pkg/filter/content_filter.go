package filter

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// ContentType 定义内容类型
type ContentType string

const (
	TextContent  ContentType = "text"
	ImageContent ContentType = "image"
	AudioContent ContentType = "audio"
	VideoContent ContentType = "video"
)

// FilterLevel 定义过滤级别
type FilterLevel int

const (
	LevelLow    FilterLevel = 1 // 低级别过滤
	LevelMedium FilterLevel = 2 // 中级别过滤
	LevelHigh   FilterLevel = 3 // 高级别过滤
)

// ContentFilter 内容过滤器接口
type ContentFilter interface {
	Filter(ctx context.Context, content string, contentType ContentType) (bool, string, error)
}

// FilterResult 过滤结果
type FilterResult struct {
	IsClean     bool            `json:"is_clean"`
	Score       float64         `json:"score"`
	Categories  map[string]bool `json:"categories"`
	Suggestions []string        `json:"suggestions"`
	Reason      string          `json:"reason"`
}

// ContentFilterService 内容过滤服务
type ContentFilterService struct {
	sensitiveWords map[string]struct{}
	regexPatterns  []*regexp.Regexp
	aiFilter       *AIFilter
	level          FilterLevel
	mu             sync.RWMutex
}

// NewContentFilterService 创建新的内容过滤服务
func NewContentFilterService(level FilterLevel) *ContentFilterService {
	return &ContentFilterService{
		sensitiveWords: make(map[string]struct{}),
		regexPatterns:  make([]*regexp.Regexp, 0),
		aiFilter:       NewAIFilter(),
		level:          level,
	}
}

// LoadSensitiveWords 加载敏感词列表
func (s *ContentFilterService) LoadSensitiveWords(words []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, word := range words {
		s.sensitiveWords[strings.ToLower(word)] = struct{}{}
	}
}

// AddRegexPattern 添加正则表达式模式
func (s *ContentFilterService) AddRegexPattern(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %v", err)
	}

	s.mu.Lock()
	s.regexPatterns = append(s.regexPatterns, regex)
	s.mu.Unlock()

	return nil
}

// Filter 过滤内容
func (s *ContentFilterService) Filter(ctx context.Context, content string, contentType ContentType) (*FilterResult, error) {
	// 1. 基础敏感词过滤
	if found, word := s.checkSensitiveWords(content); found {
		return &FilterResult{
			IsClean:    false,
			Score:      1.0,
			Categories: map[string]bool{"sensitive_words": true},
			Reason:     fmt.Sprintf("Contains sensitive word: %s", word),
		}, nil
	}

	// 2. 正则表达式匹配
	if found, pattern := s.checkRegexPatterns(content); found {
		return &FilterResult{
			IsClean:    false,
			Score:      0.8,
			Categories: map[string]bool{"pattern_match": true},
			Reason:     fmt.Sprintf("Matches forbidden pattern: %s", pattern),
		}, nil
	}

	// 3. AI 模型过滤
	aiResult, err := s.aiFilter.Analyze(ctx, content, contentType)
	if err != nil {
		return nil, fmt.Errorf("AI filter error: %v", err)
	}

	// 根据过滤级别判断
	if s.shouldBlock(aiResult) {
		return &FilterResult{
			IsClean:     false,
			Score:       aiResult.Score,
			Categories:  aiResult.Categories,
			Suggestions: aiResult.Suggestions,
			Reason:      "AI model detected inappropriate content",
		}, nil
	}

	return &FilterResult{
		IsClean:     true,
		Score:       aiResult.Score,
		Categories:  aiResult.Categories,
		Suggestions: aiResult.Suggestions,
	}, nil
}

// checkSensitiveWords 检查敏感词
func (s *ContentFilterService) checkSensitiveWords(content string) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	content = strings.ToLower(content)
	for word := range s.sensitiveWords {
		if strings.Contains(content, word) {
			return true, word
		}
	}
	return false, ""
}

// checkRegexPatterns 检查正则表达式模式
func (s *ContentFilterService) checkRegexPatterns(content string) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, pattern := range s.regexPatterns {
		if pattern.MatchString(content) {
			return true, pattern.String()
		}
	}
	return false, ""
}

// shouldBlock 根据AI分析结果和过滤级别决定是否拦截
func (s *ContentFilterService) shouldBlock(result *AIFilterResult) bool {
	switch s.level {
	case LevelLow:
		return result.Score > 0.9
	case LevelMedium:
		return result.Score > 0.7
	case LevelHigh:
		return result.Score > 0.5
	default:
		return result.Score > 0.8
	}
}
