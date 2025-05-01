package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"multi-agent-chatter/pkg/filter/model"
)

// CacheStats 缓存统计信息
type CacheStats struct {
	// 当前缓存大小
	Size int

	// 内存使用量(字节)
	MemoryUsage int64

	// 命中率
	HitRate float64

	// 平均访问时间(毫秒)
	AvgAccessTime float64

	// 过期条目数量
	ExpiredEntries int

	// 按内容类型统计
	TypeStats map[model.ContentType]TypeStats
}

// TypeStats 按内容类型的统计信息
type TypeStats struct {
	Count      int     // 条目数量
	HitRate    float64 // 命中率
	AvgLatency float64 // 平均延迟(毫秒)
}

// CacheKey 缓存键
type CacheKey struct {
	ContentType model.ContentType
	Content     string
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Value       interface{}
	Timestamp   time.Time
	AccessCount int
	Size        int64
}

// BatchResult 批量操作的结果
type BatchResult struct {
	Value interface{}
	Found bool
	Error error
}

// BatchSetItem 批量设置的条目
type BatchSetItem struct {
	ContentType model.ContentType
	Content     string
	Value       interface{}
	Size        int64
}

// BatchGetItem 批量获取的条目
type BatchGetItem struct {
	ContentType model.ContentType
	Content     string
}

// CacheManager 缓存管理器
type CacheManager struct {
	cache      map[string]CacheEntry
	mutex      sync.RWMutex
	maxEntries int
	ttl        time.Duration

	// 统计信息
	hits       int
	misses     int
	totalTime  float64
	accessTime map[string]float64

	// 监控相关
	thresholds        map[string]float64
	thresholdCallback func(CacheStats)
	evictionCallback  func(string, CacheEntry)
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(maxEntries int, ttl time.Duration) *CacheManager {
	cm := &CacheManager{
		cache:      make(map[string]CacheEntry),
		maxEntries: maxEntries,
		ttl:        ttl,
		accessTime: make(map[string]float64),
		thresholds: make(map[string]float64),
	}

	// 启动过期清理
	go cm.cleanupExpired()

	return cm
}

// cleanupExpired 定期清理过期条目
func (cm *CacheManager) cleanupExpired() {
	ticker := time.NewTicker(cm.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		cm.mutex.Lock()
		now := time.Now()
		for key, entry := range cm.cache {
			if now.Sub(entry.Timestamp) > cm.ttl {
				if cm.evictionCallback != nil {
					cm.evictionCallback(key, entry)
				}
				delete(cm.cache, key)
			}
		}
		cm.mutex.Unlock()
	}
}

// SetThreshold 设置监控阈值
func (cm *CacheManager) SetThreshold(name string, value float64) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.thresholds[name] = value
}

// SetThresholdCallback 设置阈值告警回调
func (cm *CacheManager) SetThresholdCallback(callback func(CacheStats)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.thresholdCallback = callback
}

// SetEvictionCallback 设置条目淘汰回调
func (cm *CacheManager) SetEvictionCallback(callback func(string, CacheEntry)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.evictionCallback = callback
}

// GetStats 获取缓存统计信息
func (cm *CacheManager) GetStats() CacheStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := CacheStats{
		Size:      len(cm.cache),
		TypeStats: make(map[model.ContentType]TypeStats),
	}

	var totalSize int64
	typeHits := make(map[model.ContentType]int)
	typeMisses := make(map[model.ContentType]int)
	typeLatency := make(map[model.ContentType]float64)

	// 计算各项统计
	for key, entry := range cm.cache {
		totalSize += entry.Size

		// 解析key获取内容类型
		cacheKey := cm.parseKey(key)
		if cacheKey != nil {
			typeStats := stats.TypeStats[cacheKey.ContentType]
			typeStats.Count++
			if latency, ok := cm.accessTime[key]; ok {
				typeLatency[cacheKey.ContentType] += latency
			}
			stats.TypeStats[cacheKey.ContentType] = typeStats
		}
	}

	// 计算命中率和平均访问时间
	totalAccesses := cm.hits + cm.misses
	if totalAccesses > 0 {
		stats.HitRate = float64(cm.hits) / float64(totalAccesses)
		stats.AvgAccessTime = cm.totalTime / float64(totalAccesses)
	}

	// 计算各类型的统计信息
	for contentType, typeStats := range stats.TypeStats {
		totalTypeAccesses := typeHits[contentType] + typeMisses[contentType]
		if totalTypeAccesses > 0 {
			typeStats.HitRate = float64(typeHits[contentType]) / float64(totalTypeAccesses)
			typeStats.AvgLatency = typeLatency[contentType] / float64(totalTypeAccesses)
		}
		stats.TypeStats[contentType] = typeStats
	}

	stats.MemoryUsage = totalSize

	return stats
}

// generateKey 生成缓存键
func (cm *CacheManager) generateKey(contentType model.ContentType, content string) string {
	key := CacheKey{
		ContentType: contentType,
		Content:     content,
	}

	// 使用SHA-256生成唯一键
	hasher := sha256.New()
	hasher.Write([]byte{byte(int(key.ContentType))})
	hasher.Write([]byte(key.Content))

	return hex.EncodeToString(hasher.Sum(nil))
}

// parseKey 解析缓存键
func (cm *CacheManager) parseKey(key string) *CacheKey {
	// 从key中提取内容类型和内容
	if len(key) < 2 {
		return nil
	}

	contentType := model.ContentType(key[0])
	content := key[1:]

	return &CacheKey{
		ContentType: contentType,
		Content:     content,
	}
}

// Get 获取缓存条目
func (cm *CacheManager) Get(contentType model.ContentType, content string) (interface{}, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	key := cm.generateKey(contentType, content)
	entry, ok := cm.cache[key]

	// 更新统计信息
	if ok {
		cm.hits++
		entry.AccessCount++
		cm.cache[key] = entry
	} else {
		cm.misses++
	}

	if ok && time.Since(entry.Timestamp) > cm.ttl {
		return nil, false
	}

	return entry.Value, ok
}

// Set 设置缓存条目
func (cm *CacheManager) Set(contentType model.ContentType, content string, value interface{}, size int64) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	key := cm.generateKey(contentType, content)

	// 检查缓存大小
	if len(cm.cache) >= cm.maxEntries {
		cm.evictOldest()
	}

	// 创建新条目
	entry := CacheEntry{
		Value:       value,
		Timestamp:   time.Now(),
		AccessCount: 0,
		Size:        size,
	}

	cm.cache[key] = entry
}

// evictOldest 淘汰最旧的条目
func (cm *CacheManager) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	// 找到最旧的条目
	for key, entry := range cm.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	// 如果找到了最旧的条目，删除它
	if oldestKey != "" {
		if cm.evictionCallback != nil {
			cm.evictionCallback(oldestKey, cm.cache[oldestKey])
		}
		delete(cm.cache, oldestKey)
	}
}

// Clear 清空缓存
func (cm *CacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache = make(map[string]CacheEntry)
	cm.hits = 0
	cm.misses = 0
	cm.totalTime = 0
	cm.accessTime = make(map[string]float64)
}

// BatchGet 批量获取缓存条目
func (cm *CacheManager) BatchGet(items []BatchGetItem) map[string]BatchResult {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	results := make(map[string]BatchResult)
	start := time.Now()

	for _, item := range items {
		key := cm.generateKey(item.ContentType, item.Content)
		entry, ok := cm.cache[key]

		if !ok {
			cm.misses++
			results[item.Content] = BatchResult{
				Found: false,
			}
			continue
		}

		// 检查是否过期
		if time.Since(entry.Timestamp) > cm.ttl {
			cm.misses++
			results[item.Content] = BatchResult{
				Found: false,
			}
			continue
		}

		// 更新访问统计
		cm.hits++
		entry.AccessCount++
		cm.cache[key] = entry

		results[item.Content] = BatchResult{
			Value: entry.Value,
			Found: true,
		}
	}

	// 更新访问时间统计
	elapsed := float64(time.Since(start).Milliseconds())
	cm.totalTime += elapsed
	for _, item := range items {
		key := cm.generateKey(item.ContentType, item.Content)
		cm.accessTime[key] = elapsed / float64(len(items))
	}

	return results
}

// BatchSet 批量设置缓存条目
func (cm *CacheManager) BatchSet(items []BatchSetItem) map[string]error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	results := make(map[string]error)
	now := time.Now()

	// 预先计算需要淘汰的数量
	needToEvict := len(cm.cache) + len(items) - cm.maxEntries
	if needToEvict > 0 {
		cm.evictBatch(needToEvict)
	}

	// 批量设置
	for _, item := range items {
		key := cm.generateKey(item.ContentType, item.Content)

		entry := CacheEntry{
			Value:       item.Value,
			Timestamp:   now,
			AccessCount: 0,
			Size:        item.Size,
		}

		cm.cache[key] = entry
	}

	return results
}

// evictBatch 批量淘汰指定数量的条目
func (cm *CacheManager) evictBatch(count int) {
	type evictionCandidate struct {
		key       string
		timestamp time.Time
	}

	// 收集所有条目的时间戳
	candidates := make([]evictionCandidate, 0, len(cm.cache))
	for key, entry := range cm.cache {
		candidates = append(candidates, evictionCandidate{
			key:       key,
			timestamp: entry.Timestamp,
		})
	}

	// 按时间戳排序（最旧的在前面）
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].timestamp.Before(candidates[j].timestamp)
	})

	// 淘汰指定数量的最旧条目
	for i := 0; i < count && i < len(candidates); i++ {
		key := candidates[i].key
		if cm.evictionCallback != nil {
			cm.evictionCallback(key, cm.cache[key])
		}
		delete(cm.cache, key)
	}
}
