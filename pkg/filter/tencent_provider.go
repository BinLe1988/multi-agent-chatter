package filter

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BinLe1988/multi-agent-chatter/pkg/filter/model"
)

// TencentProvider 腾讯云内容安全服务提供商
type TencentProvider struct {
	BaseProvider
	secretKey string
	region    string
	cache     *CacheManager
}

// NewTencentProvider 创建腾讯云内容安全服务提供商实例
func NewTencentProvider(config ProviderConfig) (*TencentProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Tencent API key is required")
	}
	if config.APISecret == "" {
		return nil, fmt.Errorf("Tencent API secret is required")
	}
	if config.Region == "" {
		config.Region = "ap-guangzhou" // 默认区域
	}

	return &TencentProvider{
		BaseProvider: BaseProvider{
			config: config,
			name:   "tencent",
		},
		secretKey: config.APISecret,
		region:    config.Region,
		cache:     NewCacheManager(1000, 24*time.Hour), // 24小时缓存时间，最多1000条记录
	}, nil
}

// 生成腾讯云API签名
func (p *TencentProvider) generateSignature(action, payload string) (string, string) {
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	// 1. 拼接规范请求串
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:tms.tencentcloudapi.com\n")
	signedHeaders := "content-type;host"
	hashedRequestPayload := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	// 2. 拼接待签名字符串
	algorithm := "TC3-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/tms/tc3_request", date)
	hashedCanonicalRequest := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// 3. 计算签名
	secretDate := hmacSHA256([]byte(fmt.Sprintf("TC3%s", p.secretKey)), date)
	secretService := hmacSHA256(secretDate, "tms")
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := fmt.Sprintf("%x", hmacSHA256(secretSigning, stringToSign))

	// 4. 拼接 Authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		p.config.APIKey,
		credentialScope,
		signedHeaders,
		signature)

	return authorization, fmt.Sprintf("%d", timestamp)
}

// hmacSHA256 计算HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// AnalyzeText 实现文本分析
func (p *TencentProvider) AnalyzeText(ctx context.Context, text string) (*AIFilterResult, error) {
	// 检查缓存
	if result, found := p.cache.Get(model.ContentTypeText, text); found {
		return result.(*AIFilterResult), nil
	}

	url := "https://tms.tencentcloudapi.com"

	reqBody := struct {
		Content string `json:"Content"`
	}{
		Content: base64.StdEncoding.EncodeToString([]byte(text)),
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	authorization, timestamp := p.generateSignature("TextModeration", string(reqData))

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", "TextModeration")
	req.Header.Set("X-TC-Version", "2020-12-29")
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", p.region)

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
		Response struct {
			Suggestion string   `json:"Suggestion"`
			Label      string   `json:"Label"`
			Score      float64  `json:"Score"`
			Keywords   []string `json:"Keywords"`
		} `json:"Response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 将腾讯云的分类结果转换为标准格式
	categories := make(map[string]interface{})
	categories[result.Response.Label] = result.Response.Score

	standardCategories := StandardizeCategories("tencent", categories)

	// 生成建议
	var suggestions []string
	if len(result.Response.Keywords) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Found sensitive keywords: %s", strings.Join(result.Response.Keywords, ", ")))
	}
	suggestions = append(suggestions, fmt.Sprintf("Suggestion: %s", result.Response.Suggestion))

	// 缓存结果
	filterResult := &AIFilterResult{
		Score:       result.Response.Score,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}
	p.cache.Set(model.ContentTypeText, text, filterResult, 1024) // 估算大小为1KB
	return filterResult, nil
}

// AnalyzeImage 实现图片分析
func (p *TencentProvider) AnalyzeImage(ctx context.Context, imageURL string) (*AIFilterResult, error) {
	// 检查缓存
	if result, found := p.cache.Get(model.ContentTypeImage, imageURL); found {
		return result.(*AIFilterResult), nil
	}

	url := "https://ims.tencentcloudapi.com"

	reqBody := struct {
		FileURL string `json:"FileUrl"`
	}{
		FileURL: imageURL,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	authorization, timestamp := p.generateSignature("ImageModeration", string(reqData))

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", "ImageModeration")
	req.Header.Set("X-TC-Version", "2020-12-29")
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", p.region)

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
		Response struct {
			Suggestion string `json:"Suggestion"`
			Label      string `json:"Label"`
			SubLabel   string `json:"SubLabel"`
			Score      int    `json:"Score"`
			PornInfo   *struct {
				Label    string `json:"Label"`
				Score    int    `json:"Score"`
				SubLabel string `json:"SubLabel"`
			} `json:"PornInfo"`
			TerrorismInfo *struct {
				Label    string `json:"Label"`
				Score    int    `json:"Score"`
				SubLabel string `json:"SubLabel"`
			} `json:"TerrorismInfo"`
			PoliticsInfo *struct {
				Label    string `json:"Label"`
				Score    int    `json:"Score"`
				SubLabel string `json:"SubLabel"`
			} `json:"PoliticsInfo"`
			AdsInfo *struct {
				Label    string `json:"Label"`
				Score    int    `json:"Score"`
				SubLabel string `json:"SubLabel"`
			} `json:"AdsInfo"`
		} `json:"Response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 将腾讯云的图片分析结果转换为标准格式
	categories := make(map[string]interface{})

	// 处理主分类
	categories[result.Response.Label] = float64(result.Response.Score) / 100

	// 处理色情内容
	if result.Response.PornInfo != nil && result.Response.PornInfo.Score > 0 {
		categories["Porn"] = float64(result.Response.PornInfo.Score) / 100
	}

	// 处理暴恐内容
	if result.Response.TerrorismInfo != nil && result.Response.TerrorismInfo.Score > 0 {
		categories["Terror"] = float64(result.Response.TerrorismInfo.Score) / 100
	}

	// 处理政治敏感内容
	if result.Response.PoliticsInfo != nil && result.Response.PoliticsInfo.Score > 0 {
		categories["Politics"] = float64(result.Response.PoliticsInfo.Score) / 100
	}

	// 处理广告内容
	if result.Response.AdsInfo != nil && result.Response.AdsInfo.Score > 0 {
		categories["Ad"] = float64(result.Response.AdsInfo.Score) / 100
	}

	standardCategories := StandardizeCategories("tencent", categories)

	// 生成建议
	var suggestions []string
	suggestions = append(suggestions, fmt.Sprintf("Overall category: %s", result.Response.Label))
	if result.Response.SubLabel != "" {
		suggestions = append(suggestions, fmt.Sprintf("Sub-category: %s", result.Response.SubLabel))
	}

	// 添加详细建议
	if result.Response.PornInfo != nil && result.Response.PornInfo.Score > 70 {
		suggestions = append(suggestions, fmt.Sprintf("Adult content detected: %s (%s)",
			result.Response.PornInfo.Label, result.Response.PornInfo.SubLabel))
	}
	if result.Response.TerrorismInfo != nil && result.Response.TerrorismInfo.Score > 70 {
		suggestions = append(suggestions, fmt.Sprintf("Violence content detected: %s (%s)",
			result.Response.TerrorismInfo.Label, result.Response.TerrorismInfo.SubLabel))
	}
	if result.Response.PoliticsInfo != nil && result.Response.PoliticsInfo.Score > 70 {
		suggestions = append(suggestions, fmt.Sprintf("Political content detected: %s (%s)",
			result.Response.PoliticsInfo.Label, result.Response.PoliticsInfo.SubLabel))
	}
	if result.Response.AdsInfo != nil && result.Response.AdsInfo.Score > 70 {
		suggestions = append(suggestions, fmt.Sprintf("Advertisement content detected: %s (%s)",
			result.Response.AdsInfo.Label, result.Response.AdsInfo.SubLabel))
	}

	suggestions = append(suggestions, fmt.Sprintf("Action suggestion: %s", result.Response.Suggestion))

	// 使用最高分作为综合得分
	maxScore := float64(result.Response.Score)
	if result.Response.PornInfo != nil && float64(result.Response.PornInfo.Score) > maxScore {
		maxScore = float64(result.Response.PornInfo.Score)
	}
	if result.Response.TerrorismInfo != nil && float64(result.Response.TerrorismInfo.Score) > maxScore {
		maxScore = float64(result.Response.TerrorismInfo.Score)
	}
	if result.Response.PoliticsInfo != nil && float64(result.Response.PoliticsInfo.Score) > maxScore {
		maxScore = float64(result.Response.PoliticsInfo.Score)
	}
	if result.Response.AdsInfo != nil && float64(result.Response.AdsInfo.Score) > maxScore {
		maxScore = float64(result.Response.AdsInfo.Score)
	}

	// 缓存结果
	filterResult := &AIFilterResult{
		Score:       maxScore / 100,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}
	p.cache.Set(model.ContentTypeImage, imageURL, filterResult, 1024)
	return filterResult, nil
}

// AnalyzeAudio 实现音频分析
func (p *TencentProvider) AnalyzeAudio(ctx context.Context, audioURL string) (*AIFilterResult, error) {
	// 检查缓存
	if result, found := p.cache.Get(model.ContentTypeAudio, audioURL); found {
		return result.(*AIFilterResult), nil
	}

	url := "https://ams.tencentcloudapi.com"

	reqBody := struct {
		Tasks []struct {
			URL string `json:"Url"`
		} `json:"Tasks"`
		BizType string `json:"BizType"`
	}{
		Tasks: []struct {
			URL string `json:"Url"`
		}{{URL: audioURL}},
		BizType: "default",
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	authorization, timestamp := p.generateSignature("CreateAudioModerationTask", string(reqData))

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", "CreateAudioModerationTask")
	req.Header.Set("X-TC-Version", "2020-12-29")
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", p.region)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var createResult struct {
		Response struct {
			Results []struct {
				TaskID string `json:"TaskId"`
				Code   string `json:"Code"`
			} `json:"Results"`
		} `json:"Response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		return nil, fmt.Errorf("failed to decode create task response: %v", err)
	}

	if len(createResult.Response.Results) == 0 {
		return nil, fmt.Errorf("no task created")
	}

	taskID := createResult.Response.Results[0].TaskID
	if taskID == "" {
		return nil, fmt.Errorf("invalid task ID")
	}

	// 等待任务完成并获取结果
	var result struct {
		Response struct {
			TaskID     string `json:"TaskId"`
			DataID     string `json:"DataId"`
			Status     string `json:"Status"`
			Suggestion string `json:"Suggestion"`
			Labels     []struct {
				Label    string `json:"Label"`
				Score    int    `json:"Score"`
				SubLabel string `json:"SubLabel"`
			} `json:"Labels"`
			AudioText string `json:"AudioText"`
		} `json:"Response"`
	}

	// 轮询检查任务状态
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(2 * time.Second)

		// 构建查询请求
		queryBody := struct {
			TaskID string `json:"TaskId"`
		}{
			TaskID: taskID,
		}

		queryData, err := json.Marshal(queryBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query request: %v", err)
		}

		authorization, timestamp = p.generateSignature("DescribeTaskDetail", string(queryData))

		queryReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(queryData)))
		if err != nil {
			return nil, fmt.Errorf("failed to create query request: %v", err)
		}

		queryReq.Header.Set("Content-Type", "application/json")
		queryReq.Header.Set("Authorization", authorization)
		queryReq.Header.Set("X-TC-Action", "DescribeTaskDetail")
		queryReq.Header.Set("X-TC-Version", "2020-12-29")
		queryReq.Header.Set("X-TC-Timestamp", timestamp)
		queryReq.Header.Set("X-TC-Region", p.region)

		queryResp, err := client.Do(queryReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send query request: %v", err)
		}

		if err := json.NewDecoder(queryResp.Body).Decode(&result); err != nil {
			queryResp.Body.Close()
			return nil, fmt.Errorf("failed to decode query response: %v", err)
		}
		queryResp.Body.Close()

		if result.Response.Status == "Success" {
			break
		}

		if result.Response.Status == "Failed" {
			return nil, fmt.Errorf("task failed")
		}

		if i == maxRetries-1 {
			return nil, fmt.Errorf("task timeout")
		}
	}

	// 将腾讯云的音频分析结果转换为标准格式
	categories := make(map[string]interface{})
	for _, label := range result.Response.Labels {
		categories[label.Label] = float64(label.Score) / 100
	}

	standardCategories := StandardizeCategories("tencent", categories)

	// 生成建议
	var suggestions []string
	if result.Response.AudioText != "" {
		suggestions = append(suggestions, fmt.Sprintf("Transcribed text: %s", result.Response.AudioText))
	}

	for _, label := range result.Response.Labels {
		if label.Score > 70 {
			suggestions = append(suggestions, fmt.Sprintf("Detected %s content (%s)",
				label.Label, label.SubLabel))
		}
	}

	suggestions = append(suggestions, fmt.Sprintf("Action suggestion: %s", result.Response.Suggestion))

	// 计算综合得分
	var maxScore float64
	for _, label := range result.Response.Labels {
		if float64(label.Score) > maxScore {
			maxScore = float64(label.Score)
		}
	}

	// 缓存结果
	filterResult := &AIFilterResult{
		Score:       maxScore / 100,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}
	p.cache.Set(model.ContentTypeAudio, audioURL, filterResult, 1024)
	return filterResult, nil
}

// AnalyzeVideo 实现视频分析
func (p *TencentProvider) AnalyzeVideo(ctx context.Context, videoURL string) (*AIFilterResult, error) {
	// 检查缓存
	if result, found := p.cache.Get(model.ContentTypeVideo, videoURL); found {
		return result.(*AIFilterResult), nil
	}

	url := "https://vms.tencentcloudapi.com"

	reqBody := struct {
		VideoURL string `json:"VideoUrl"`
		BizType  string `json:"BizType"`
	}{
		VideoURL: videoURL,
		BizType:  "default",
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	authorization, timestamp := p.generateSignature("CreateVideoModerationTask", string(reqData))

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", "CreateVideoModerationTask")
	req.Header.Set("X-TC-Version", "2020-12-29")
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", p.region)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var createResult struct {
		Response struct {
			TaskID string `json:"TaskId"`
		} `json:"Response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		return nil, fmt.Errorf("failed to decode create task response: %v", err)
	}

	if createResult.Response.TaskID == "" {
		return nil, fmt.Errorf("invalid task ID")
	}

	// 等待任务完成并获取结果
	var result struct {
		Response struct {
			Status     string `json:"Status"`
			Suggestion string `json:"Suggestion"`
			Labels     []struct {
				Label    string `json:"Label"`
				Score    int    `json:"Score"`
				SubLabel string `json:"SubLabel"`
			} `json:"Labels"`
			ImageResults []struct {
				Suggestion string `json:"Suggestion"`
				Label      string `json:"Label"`
				SubLabel   string `json:"SubLabel"`
				Score      int    `json:"Score"`
				Timestamp  int64  `json:"Timestamp"`
			} `json:"ImageResults"`
			AudioResults []struct {
				Suggestion string `json:"Suggestion"`
				Label      string `json:"Label"`
				SubLabel   string `json:"SubLabel"`
				Score      int    `json:"Score"`
				Text       string `json:"Text"`
				StartTime  int64  `json:"StartTime"`
				EndTime    int64  `json:"EndTime"`
			} `json:"AudioResults"`
		} `json:"Response"`
	}

	// 轮询检查任务状态
	maxRetries := 30 // 视频处理可能需要更长时间
	for i := 0; i < maxRetries; i++ {
		time.Sleep(5 * time.Second) // 视频处理间隔更长

		// 构建查询请求
		queryBody := struct {
			TaskID string `json:"TaskId"`
		}{
			TaskID: createResult.Response.TaskID,
		}

		queryData, err := json.Marshal(queryBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query request: %v", err)
		}

		authorization, timestamp = p.generateSignature("DescribeTaskDetail", string(queryData))

		queryReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(queryData)))
		if err != nil {
			return nil, fmt.Errorf("failed to create query request: %v", err)
		}

		queryReq.Header.Set("Content-Type", "application/json")
		queryReq.Header.Set("Authorization", authorization)
		queryReq.Header.Set("X-TC-Action", "DescribeTaskDetail")
		queryReq.Header.Set("X-TC-Version", "2020-12-29")
		queryReq.Header.Set("X-TC-Timestamp", timestamp)
		queryReq.Header.Set("X-TC-Region", p.region)

		queryResp, err := client.Do(queryReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send query request: %v", err)
		}

		if err := json.NewDecoder(queryResp.Body).Decode(&result); err != nil {
			queryResp.Body.Close()
			return nil, fmt.Errorf("failed to decode query response: %v", err)
		}
		queryResp.Body.Close()

		if result.Response.Status == "Success" {
			break
		}

		if result.Response.Status == "Failed" {
			return nil, fmt.Errorf("task failed")
		}

		if i == maxRetries-1 {
			return nil, fmt.Errorf("task timeout")
		}
	}

	// 将腾讯云的视频分析结果转换为标准格式
	categories := make(map[string]interface{})

	// 处理总体标签
	for _, label := range result.Response.Labels {
		categories[label.Label] = float64(label.Score) / 100
	}

	standardCategories := StandardizeCategories("tencent", categories)

	// 生成建议
	var suggestions []string

	// 添加总体建议
	suggestions = append(suggestions, fmt.Sprintf("Overall suggestion: %s", result.Response.Suggestion))

	// 处理视频帧结果
	if len(result.Response.ImageResults) > 0 {
		suggestions = append(suggestions, "\nSuspicious video frames:")
		for _, frame := range result.Response.ImageResults {
			if frame.Score > 70 {
				timeStr := time.Unix(frame.Timestamp, 0).Format("15:04:05")
				suggestions = append(suggestions, fmt.Sprintf("- At %s: Detected %s content (%s) with score %d%%",
					timeStr, frame.Label, frame.SubLabel, frame.Score))
			}
		}
	}

	// 处理音频结果
	if len(result.Response.AudioResults) > 0 {
		suggestions = append(suggestions, "\nSuspicious audio segments:")
		for _, audio := range result.Response.AudioResults {
			if audio.Score > 70 {
				startTime := time.Unix(audio.StartTime, 0).Format("15:04:05")
				endTime := time.Unix(audio.EndTime, 0).Format("15:04:05")
				suggestions = append(suggestions, fmt.Sprintf("- From %s to %s: Detected %s content (%s)",
					startTime, endTime, audio.Label, audio.SubLabel))
				if audio.Text != "" {
					suggestions = append(suggestions, fmt.Sprintf("  Text: %s", audio.Text))
				}
			}
		}
	}

	// 计算综合得分
	var maxScore float64

	// 检查总体标签得分
	for _, label := range result.Response.Labels {
		if float64(label.Score) > maxScore {
			maxScore = float64(label.Score)
		}
	}

	// 检查视频帧得分
	for _, frame := range result.Response.ImageResults {
		if float64(frame.Score) > maxScore {
			maxScore = float64(frame.Score)
		}
	}

	// 检查音频片段得分
	for _, audio := range result.Response.AudioResults {
		if float64(audio.Score) > maxScore {
			maxScore = float64(audio.Score)
		}
	}

	// 缓存结果
	filterResult := &AIFilterResult{
		Score:       maxScore / 100,
		Categories:  standardCategories,
		Suggestions: suggestions,
	}
	p.cache.Set(model.ContentTypeVideo, videoURL, filterResult, 1024)
	return filterResult, nil
}
