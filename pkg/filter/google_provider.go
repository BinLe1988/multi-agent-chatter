package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GoogleProvider Google Cloud Content Safety API服务提供商
type GoogleProvider struct {
	BaseProvider
	projectID string
}

// NewGoogleProvider 创建Google Cloud Content Safety API服务提供商实例
func NewGoogleProvider(config ProviderConfig) (*GoogleProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Google API key is required")
	}
	if config.Region == "" {
		return nil, fmt.Errorf("Google project ID is required")
	}

	return &GoogleProvider{
		BaseProvider: BaseProvider{
			config: config,
			name:   "google",
		},
		projectID: config.Region, // 使用Region字段存储projectID
	}, nil
}

// AnalyzeText 实现文本分析
func (p *GoogleProvider) AnalyzeText(ctx context.Context, text string) (*AIFilterResult, error) {
	url := fmt.Sprintf("https://contentthreat.googleapis.com/v1beta1/projects/%s/locations/global/text:analyze", p.projectID)

	reqBody := struct {
		Text string `json:"text"`
	}{
		Text: text,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Categories []struct {
			Name       string  `json:"name"`
			Confidence float64 `json:"confidence"`
		} `json:"categories"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 将Google的分类结果转换为标准格式
	categories := make(map[string]interface{})
	for _, category := range result.Categories {
		categories[category.Name] = category.Confidence
	}

	standardCategories := StandardizeCategories("google", categories)

	// 生成建议
	var suggestions []string
	for _, category := range result.Categories {
		if category.Confidence > 0.7 {
			suggestions = append(suggestions, fmt.Sprintf("High confidence (%0.2f) of %s content", category.Confidence, category.Name))
		}
	}

	// 计算综合得分
	var maxScore float64
	for _, category := range result.Categories {
		if category.Confidence > maxScore {
			maxScore = category.Confidence
		}
	}

	return &AIFilterResult{
		Score:       maxScore,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}, nil
}

// AnalyzeImage 实现图片分析
func (p *GoogleProvider) AnalyzeImage(ctx context.Context, imageURL string) (*AIFilterResult, error) {
	url := fmt.Sprintf("https://contentthreat.googleapis.com/v1beta1/projects/%s/locations/global/image:analyze", p.projectID)

	reqBody := struct {
		ImageSource struct {
			ImageURI string `json:"imageUri"`
		} `json:"imageSource"`
	}{
		ImageSource: struct {
			ImageURI string `json:"imageUri"`
		}{
			ImageURI: imageURL,
		},
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Categories []struct {
			Name       string  `json:"name"`
			Confidence float64 `json:"confidence"`
		} `json:"categories"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 将Google的图片分析结果转换为标准格式
	categories := make(map[string]interface{})
	for _, category := range result.Categories {
		categories[category.Name] = category.Confidence
	}

	standardCategories := StandardizeCategories("google", categories)

	// 生成建议
	var suggestions []string
	for _, category := range result.Categories {
		if category.Confidence > 0.7 {
			suggestions = append(suggestions, fmt.Sprintf("High confidence (%0.2f) of %s content in image", category.Confidence, category.Name))
		}
	}

	// 计算综合得分
	var maxScore float64
	for _, category := range result.Categories {
		if category.Confidence > maxScore {
			maxScore = category.Confidence
		}
	}

	return &AIFilterResult{
		Score:       maxScore,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}, nil
}

// AnalyzeAudio 实现音频分析
func (p *GoogleProvider) AnalyzeAudio(ctx context.Context, audioURL string) (*AIFilterResult, error) {
	// TODO: 实现Google音频内容分析
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}

// AnalyzeVideo 实现视频分析
func (p *GoogleProvider) AnalyzeVideo(ctx context.Context, videoURL string) (*AIFilterResult, error) {
	// TODO: 实现Google视频内容分析
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}
