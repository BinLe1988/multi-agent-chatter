package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/BinLe1988/multi-agent-chatter/pkg/filter/model"
)

// AIFilterResult AI过滤结果
type AIFilterResult struct {
	Score       float64         `json:"score"`
	Categories  map[string]bool `json:"categories"`
	Suggestions []string        `json:"suggestions"`
}

// AIFilter AI内容过滤器
type AIFilter struct {
	apiKey     string
	apiBaseURL string
	httpClient *http.Client
}

// NewAIFilter 创建新的AI过滤器
func NewAIFilter() *AIFilter {
	return &AIFilter{
		apiKey:     "", // TODO: 从配置中加载
		apiBaseURL: "https://api.openai.com/v1/moderations",
		httpClient: &http.Client{},
	}
}

// SetAPIKey 设置API密钥
func (f *AIFilter) SetAPIKey(apiKey string) {
	f.apiKey = apiKey
}

// Analyze 分析内容
func (f *AIFilter) Analyze(ctx context.Context, content string, contentType model.ContentType) (*AIFilterResult, error) {
	switch contentType {
	case model.ContentTypeText:
		return f.analyzeText(ctx, content)
	case model.ContentTypeImage:
		return f.analyzeImage(ctx, content)
	case model.ContentTypeAudio:
		return f.analyzeAudio(ctx, content)
	case model.ContentTypeVideo:
		return f.analyzeVideo(ctx, content)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// analyzeText 分析文本内容
func (f *AIFilter) analyzeText(ctx context.Context, text string) (*AIFilterResult, error) {
	// 使用OpenAI的Moderation API进行内容审核
	type ModerationRequest struct {
		Input string `json:"input"`
	}

	type ModerationResponse struct {
		Results []struct {
			Flagged        bool               `json:"flagged"`
			Categories     map[string]bool    `json:"categories"`
			CategoryScores map[string]float64 `json:"category_scores"`
		} `json:"results"`
	}

	reqBody, err := json.Marshal(ModerationRequest{Input: text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.apiBaseURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.apiKey)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var moderationResp ModerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&moderationResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(moderationResp.Results) == 0 {
		return nil, fmt.Errorf("no moderation results returned")
	}

	result := moderationResp.Results[0]

	// 计算综合得分
	var maxScore float64
	for _, score := range result.CategoryScores {
		if score > maxScore {
			maxScore = score
		}
	}

	// 生成建议
	var suggestions []string
	for category, flagged := range result.Categories {
		if flagged {
			suggestions = append(suggestions, fmt.Sprintf("Content contains inappropriate %s", category))
		}
	}

	return &AIFilterResult{
		Score:       maxScore,
		Categories:  result.Categories,
		Suggestions: suggestions,
	}, nil
}

// analyzeImage 分析图片内容
func (f *AIFilter) analyzeImage(ctx context.Context, imageURL string) (*AIFilterResult, error) {
	// TODO: 实现图片内容分析
	// 可以使用其他AI服务，如Google Cloud Vision API或Azure Computer Vision
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}

// analyzeAudio 分析音频内容
func (f *AIFilter) analyzeAudio(ctx context.Context, audioURL string) (*AIFilterResult, error) {
	// TODO: 实现音频内容分析
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}

// analyzeVideo 分析视频内容
func (f *AIFilter) analyzeVideo(ctx context.Context, videoURL string) (*AIFilterResult, error) {
	// TODO: 实现视频内容分析
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}
