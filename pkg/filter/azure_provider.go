package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AzureProvider Azure Content Moderator服务提供商
type AzureProvider struct {
	BaseProvider
	endpoint string
}

// NewAzureProvider 创建Azure Content Moderator服务提供商实例
func NewAzureProvider(config ProviderConfig) (*AzureProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Azure API key is required")
	}
	if config.Endpoint == "" {
		return nil, fmt.Errorf("Azure endpoint is required")
	}

	return &AzureProvider{
		BaseProvider: BaseProvider{
			config: config,
			name:   "azure",
		},
		endpoint: config.Endpoint,
	}, nil
}

// AnalyzeText 实现文本分析
func (p *AzureProvider) AnalyzeText(ctx context.Context, text string) (*AIFilterResult, error) {
	url := fmt.Sprintf("%s/contentmoderator/moderate/v1.0/ProcessText/Screen", p.endpoint)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Ocp-Apim-Subscription-Key", p.config.APIKey)

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
		Classification struct {
			Category1 struct {
				Score float64 `json:"Score"`
			} `json:"Category1"`
			Category2 struct {
				Score float64 `json:"Score"`
			} `json:"Category2"`
			Category3 struct {
				Score float64 `json:"Score"`
			} `json:"Category3"`
		} `json:"Classification"`
		Terms []struct {
			Term string `json:"Term"`
		} `json:"Terms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 将Azure的分类结果转换为标准格式
	categories := make(map[string]interface{})
	if result.Classification.Category1.Score > 0.5 {
		categories["Adult"] = result.Classification.Category1.Score
	}
	if result.Classification.Category2.Score > 0.5 {
		categories["Violence"] = result.Classification.Category2.Score
	}
	if result.Classification.Category3.Score > 0.5 {
		categories["Hate"] = result.Classification.Category3.Score
	}

	standardCategories := StandardizeCategories("azure", categories)

	// 生成建议
	var suggestions []string
	for term := range result.Terms {
		suggestions = append(suggestions, fmt.Sprintf("Found inappropriate term: %s", term))
	}

	// 计算综合得分
	maxScore := 0.0
	for _, score := range []float64{
		result.Classification.Category1.Score,
		result.Classification.Category2.Score,
		result.Classification.Category3.Score,
	} {
		if score > maxScore {
			maxScore = score
		}
	}

	return &AIFilterResult{
		Score:       maxScore,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}, nil
}

// AnalyzeImage 实现图片分析
func (p *AzureProvider) AnalyzeImage(ctx context.Context, imageURL string) (*AIFilterResult, error) {
	url := fmt.Sprintf("%s/contentmoderator/moderate/v1.0/ProcessImage/Evaluate", p.endpoint)

	reqBody := struct {
		DataRepresentation string `json:"DataRepresentation"`
		Value              string `json:"Value"`
	}{
		DataRepresentation: "URL",
		Value:              imageURL,
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
	req.Header.Set("Ocp-Apim-Subscription-Key", p.config.APIKey)

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
		AdultClassificationScore float64 `json:"AdultClassificationScore"`
		RacyClassificationScore  float64 `json:"RacyClassificationScore"`
		IsImageAdultClassified   bool    `json:"IsImageAdultClassified"`
		IsImageRacyClassified    bool    `json:"IsImageRacyClassified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 将Azure的图片分析结果转换为标准格式
	categories := make(map[string]interface{})
	if result.IsImageAdultClassified {
		categories["Adult"] = result.AdultClassificationScore
	}
	if result.IsImageRacyClassified {
		categories["Racy"] = result.RacyClassificationScore
	}

	standardCategories := StandardizeCategories("azure", categories)

	// 生成建议
	var suggestions []string
	if result.IsImageAdultClassified {
		suggestions = append(suggestions, "Image contains adult content")
	}
	if result.IsImageRacyClassified {
		suggestions = append(suggestions, "Image contains racy content")
	}

	// 使用最高分作为综合得分
	maxScore := result.AdultClassificationScore
	if result.RacyClassificationScore > maxScore {
		maxScore = result.RacyClassificationScore
	}

	return &AIFilterResult{
		Score:       maxScore,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}, nil
}

// AnalyzeAudio 实现音频分析
func (p *AzureProvider) AnalyzeAudio(ctx context.Context, audioURL string) (*AIFilterResult, error) {
	// TODO: 实现Azure音频内容分析
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}

// AnalyzeVideo 实现视频分析
func (p *AzureProvider) AnalyzeVideo(ctx context.Context, videoURL string) (*AIFilterResult, error) {
	// TODO: 实现Azure视频内容分析
	return &AIFilterResult{
		Score:      0.0,
		Categories: make(map[string]bool),
	}, nil
}
