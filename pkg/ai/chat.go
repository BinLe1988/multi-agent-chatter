package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"multi-agent-chatter/config"
	"net/http"
)

// AI相关配置
var (
	apiKey string
	model  string
)

// 初始化AI配置
func InitConfig(cfg config.AI) {
	apiKey = cfg.APIKey
	model = cfg.Model
}

// 请求结构体
type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 响应结构体
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateResponse 生成AI回复
func GenerateResponse(userMessage string, sessionContext string) (string, error) {
	if apiKey == "" {
		return "AI服务未配置，请联系管理员。", nil
	}

	// 构建请求
	messages := []message{
		{Role: "system", Content: "你是一个智能助手，请简明扼要地回答问题。"},
		{Role: "user", Content: userMessage},
	}

	requestBody := openAIRequest{
		Model:    model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if response.Error.Message != "" {
		return "", errors.New(response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("no response from AI")
	}

	return response.Choices[0].Message.Content, nil
}
