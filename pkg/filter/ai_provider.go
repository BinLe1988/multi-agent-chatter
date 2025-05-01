package filter

import (
	"context"
	"fmt"
)

// AIProvider 定义AI服务提供商接口
type AIProvider interface {
	// AnalyzeText 分析文本内容
	AnalyzeText(ctx context.Context, text string) (*AIFilterResult, error)
	// AnalyzeImage 分析图片内容
	AnalyzeImage(ctx context.Context, imageURL string) (*AIFilterResult, error)
	// AnalyzeAudio 分析音频内容
	AnalyzeAudio(ctx context.Context, audioURL string) (*AIFilterResult, error)
	// AnalyzeVideo 分析视频内容
	AnalyzeVideo(ctx context.Context, videoURL string) (*AIFilterResult, error)
	// Name 获取提供商名称
	Name() string
}

// ProviderType 定义AI服务提供商类型
type ProviderType string

const (
	ProviderOpenAI  ProviderType = "openai"
	ProviderAzure   ProviderType = "azure"
	ProviderGoogle  ProviderType = "google"
	ProviderTencent ProviderType = "tencent"
)

// ProviderConfig AI服务提供商配置
type ProviderConfig struct {
	Type      ProviderType `json:"type"`
	APIKey    string       `json:"api_key"`
	APISecret string       `json:"api_secret,omitempty"`
	Region    string       `json:"region,omitempty"`
	Endpoint  string       `json:"endpoint,omitempty"`
}

// NewAIProvider 创建AI服务提供商实例
func NewAIProvider(config ProviderConfig) (AIProvider, error) {
	switch config.Type {
	case ProviderOpenAI:
		return NewOpenAIProvider(config)
	case ProviderAzure:
		return NewAzureProvider(config)
	case ProviderGoogle:
		return NewGoogleProvider(config)
	case ProviderTencent:
		return NewTencentProvider(config)
	default:
		return nil, fmt.Errorf("unsupported AI provider type: %s", config.Type)
	}
}

// BaseProvider 基础AI服务提供商实现
type BaseProvider struct {
	config ProviderConfig
	name   string
}

func (p *BaseProvider) Name() string {
	return p.name
}

// CategoryMapping 定义不同提供商的类别映射
type CategoryMapping struct {
	ProviderCategory string
	StandardCategory string
	Severity         float64
}

// StandardizeCategories 标准化分类结果
func StandardizeCategories(providerName string, categories map[string]interface{}) map[string]bool {
	// 定义标准化分类映射
	mappings := map[string][]CategoryMapping{
		"openai": {
			{"hate", "hate_speech", 1.0},
			{"sexual", "adult_content", 1.0},
			{"violence", "violence", 1.0},
			{"self-harm", "self_harm", 1.0},
		},
		"azure": {
			{"Adult", "adult_content", 1.0},
			{"Violence", "violence", 1.0},
			{"Hate", "hate_speech", 1.0},
			{"SelfHarm", "self_harm", 1.0},
		},
		"google": {
			{"adult", "adult_content", 1.0},
			{"violence", "violence", 1.0},
			{"hate", "hate_speech", 1.0},
			{"harassment", "harassment", 1.0},
		},
		"tencent": {
			{"Porn", "adult_content", 1.0},
			{"Terror", "violence", 1.0},
			{"Ad", "advertisement", 0.5},
			{"Abuse", "hate_speech", 1.0},
		},
	}

	result := make(map[string]bool)
	if providerMappings, ok := mappings[providerName]; ok {
		for _, mapping := range providerMappings {
			if val, exists := categories[mapping.ProviderCategory]; exists {
				if score, ok := val.(float64); ok && score >= mapping.Severity {
					result[mapping.StandardCategory] = true
				}
			}
		}
	}
	return result
}
