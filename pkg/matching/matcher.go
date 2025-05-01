package matching

import (
	"math"
	"sort"

	"multi-agent-chatter/models"
)

// MatchScore 表示匹配分数
type MatchScore struct {
	UserID uint
	Score  float64
}

// Matcher 智能匹配器
type Matcher struct {
	// 权重配置
	weights struct {
		interests   float64
		tags        float64
		activeTime  float64
		location    float64
		interaction float64
	}
}

// NewMatcher 创建新的匹配器实例
func NewMatcher() *Matcher {
	m := &Matcher{}
	// 设置默认权重
	m.weights.interests = 0.3
	m.weights.tags = 0.2
	m.weights.activeTime = 0.2
	m.weights.location = 0.15
	m.weights.interaction = 0.15
	return m
}

// Match 执行智能匹配
func (m *Matcher) Match(user *models.UserProfile, candidates []*models.UserProfile) []MatchScore {
	var scores []MatchScore

	for _, candidate := range candidates {
		if candidate.ID == user.ID {
			continue
		}

		score := m.calculateMatchScore(user, candidate)
		scores = append(scores, MatchScore{
			UserID: candidate.ID,
			Score:  score,
		})
	}

	// 按分数降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores
}

// calculateMatchScore 计算两个用户之间的匹配分数
func (m *Matcher) calculateMatchScore(user1, user2 *models.UserProfile) float64 {
	score := 0.0

	// 1. 兴趣相似度
	interestScore := m.calculateInterestSimilarity(user1.Interests, user2.Interests)
	score += interestScore * m.weights.interests

	// 2. 标签相似度
	tagScore := m.calculateTagSimilarity(user1.Tags, user2.Tags)
	score += tagScore * m.weights.tags

	// 3. 活跃时间重叠度
	activeTimeScore := m.calculateActiveTimeOverlap(user1.ActiveHours, user2.ActiveHours)
	score += activeTimeScore * m.weights.activeTime

	// 4. 地理位置接近度
	locationScore := m.calculateLocationSimilarity(user1.Location, user2.Location)
	score += locationScore * m.weights.location

	// 5. 互动活跃度匹配
	interactionScore := m.calculateInteractionCompatibility(user1.InteractionScore, user2.InteractionScore)
	score += interactionScore * m.weights.interaction

	return score
}

// calculateInterestSimilarity 计算兴趣相似度
func (m *Matcher) calculateInterestSimilarity(interests1, interests2 []models.Interest) float64 {
	if len(interests1) == 0 || len(interests2) == 0 {
		return 0
	}

	// 使用余弦相似度计算兴趣向量的相似度
	interestMap1 := make(map[string]float64)
	interestMap2 := make(map[string]float64)

	for _, interest := range interests1 {
		interestMap1[interest.Name] = interest.Score
	}

	for _, interest := range interests2 {
		interestMap2[interest.Name] = interest.Score
	}

	return calculateCosineSimilarity(interestMap1, interestMap2)
}

// calculateTagSimilarity 计算标签相似度
func (m *Matcher) calculateTagSimilarity(tags1, tags2 []models.Tag) float64 {
	if len(tags1) == 0 || len(tags2) == 0 {
		return 0
	}

	tagMap1 := make(map[string]float64)
	tagMap2 := make(map[string]float64)

	for _, tag := range tags1 {
		tagMap1[tag.Name] = tag.Weight
	}

	for _, tag := range tags2 {
		tagMap2[tag.Name] = tag.Weight
	}

	return calculateCosineSimilarity(tagMap1, tagMap2)
}

// calculateActiveTimeOverlap 计算活跃时间重叠度
func (m *Matcher) calculateActiveTimeOverlap(hours1, hours2 []int) float64 {
	if len(hours1) == 0 || len(hours2) == 0 {
		return 0
	}

	// 转换为集合便于计算交集
	set1 := make(map[int]bool)
	for _, h := range hours1 {
		set1[h] = true
	}

	overlap := 0
	for _, h := range hours2 {
		if set1[h] {
			overlap++
		}
	}

	// 计算Jaccard相似度
	union := len(hours1) + len(hours2) - overlap
	if union == 0 {
		return 0
	}

	return float64(overlap) / float64(union)
}

// calculateLocationSimilarity 计算地理位置相似度
func (m *Matcher) calculateLocationSimilarity(loc1, loc2 string) float64 {
	if loc1 == "" || loc2 == "" {
		return 0
	}

	// 简单实现：相同返回1，不同返回0
	// TODO: 可以扩展为使用地理编码和距离计算
	if loc1 == loc2 {
		return 1
	}
	return 0
}

// calculateInteractionCompatibility 计算互动兼容度
func (m *Matcher) calculateInteractionCompatibility(score1, score2 float64) float64 {
	// 使用高斯函数计算分数差异，差异越小分数越高
	diff := math.Abs(score1 - score2)
	return math.Exp(-(diff * diff) / 2)
}

// calculateCosineSimilarity 计算余弦相似度
func calculateCosineSimilarity(v1, v2 map[string]float64) float64 {
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	// 计算点积和向量范数
	for k, val1 := range v1 {
		if val2, ok := v2[k]; ok {
			dotProduct += val1 * val2
		}
		norm1 += val1 * val1
	}

	for _, val2 := range v2 {
		norm2 += val2 * val2
	}

	// 避免除零错误
	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}
