package filter

import (
	"fmt"
	"testing"
	"time"

	"multi-agent-chatter/pkg/filter/model"

	"github.com/stretchr/testify/assert"
)

func TestBatchGet(t *testing.T) {
	cache := NewCacheManager(10, time.Hour)

	// 预先设置一些数据
	items := []BatchSetItem{
		{model.ContentTypeText, "key1", "value1", 100},
		{model.ContentTypeImage, "key2", "value2", 200},
		{model.ContentTypeAudio, "key3", "value3", 300},
		{model.ContentTypeVideo, "key4", "value4", 400},
	}
	cache.BatchSet(items)

	// 测试批量获取
	getItems := []BatchGetItem{
		{model.ContentTypeText, "key1"},
		{model.ContentTypeImage, "key2"},
		{model.ContentTypeAudio, "key3"},
		{model.ContentTypeVideo, "key4"},
		{model.ContentTypeText, "nonexistent"}, // 不存在的键
	}

	results := cache.BatchGet(getItems)

	// 验证结果
	assert.Equal(t, 5, len(results))

	// 验证存在的键
	for i := 0; i < 4; i++ {
		result := results[getItems[i].Content]
		assert.True(t, result.Found)
		assert.Equal(t, items[i].Value, result.Value)
		assert.Nil(t, result.Error)
	}

	// 验证不存在的键
	result := results["nonexistent"]
	assert.False(t, result.Found)
	assert.Nil(t, result.Value)
	assert.Nil(t, result.Error)

	// 验证统计信息
	stats := cache.GetStats()
	assert.Equal(t, 4, stats.Hits)
	assert.Equal(t, 1, stats.Misses)
	assert.True(t, stats.AvgAccessTime > 0)
}

func TestBatchSet(t *testing.T) {
	cache := NewCacheManager(5, time.Hour)

	// 测试批量设置
	items := []BatchSetItem{
		{model.ContentTypeText, "key1", "value1", 100},
		{model.ContentTypeImage, "key2", "value2", 200},
		{model.ContentTypeAudio, "key3", "value3", 300},
		{model.ContentTypeVideo, "key4", "value4", 400},
		{model.ContentTypeText, "key5", "value5", 500},
		{model.ContentTypeImage, "key6", "value6", 600}, // 超出容量
	}

	// 记录被淘汰的条目
	var evicted []string
	cache.SetEvictionCallback(func(key string, entry CacheEntry) {
		evicted = append(evicted, key)
	})

	results := cache.BatchSet(items)
	assert.Empty(t, results) // 没有错误

	// 验证缓存大小
	stats := cache.GetStats()
	assert.Equal(t, 5, stats.Size) // 最大容量为5

	// 验证被淘汰的条目
	assert.Equal(t, 1, len(evicted)) // 应该有一个条目被淘汰

	// 验证最新的条目仍在缓存中
	result := cache.BatchGet([]BatchGetItem{{model.ContentTypeImage, "key6"}})
	assert.True(t, result["key6"].Found)

	// 验证内存使用
	assert.True(t, stats.MemoryUsage > 0)
}

func TestBatchOperationsWithExpiry(t *testing.T) {
	cache := NewCacheManager(10, 100*time.Millisecond)

	// 设置测试数据
	items := []BatchSetItem{
		{model.ContentTypeText, "key1", "value1", 100},
		{model.ContentTypeImage, "key2", "value2", 200},
	}
	cache.BatchSet(items)

	// 验证数据已设置
	results := cache.BatchGet([]BatchGetItem{
		{model.ContentTypeText, "key1"},
		{model.ContentTypeImage, "key2"},
	})
	assert.True(t, results["key1"].Found)
	assert.True(t, results["key2"].Found)

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 验证数据已过期
	results = cache.BatchGet([]BatchGetItem{
		{model.ContentTypeText, "key1"},
		{model.ContentTypeImage, "key2"},
	})
	assert.False(t, results["key1"].Found)
	assert.False(t, results["key2"].Found)

	// 验证统计信息
	stats := cache.GetStats()
	assert.Equal(t, 2, stats.Hits)   // 第一次获取成功
	assert.Equal(t, 2, stats.Misses) // 第二次获取失败
}

func TestBatchSetWithEviction(t *testing.T) {
	cache := NewCacheManager(3, time.Hour)

	// 记录被淘汰的条目
	var evictedKeys []string
	cache.SetEvictionCallback(func(key string, entry CacheEntry) {
		evictedKeys = append(evictedKeys, key)
	})

	// 第一批设置（填满缓存）
	firstBatch := []BatchSetItem{
		{model.ContentTypeText, "key1", "value1", 100},
		{model.ContentTypeImage, "key2", "value2", 200},
		{model.ContentTypeAudio, "key3", "value3", 300},
	}
	cache.BatchSet(firstBatch)

	// 验证缓存已满
	stats := cache.GetStats()
	assert.Equal(t, 3, stats.Size)

	// 第二批设置（触发淘汰）
	secondBatch := []BatchSetItem{
		{model.ContentTypeVideo, "key4", "value4", 400},
		{model.ContentTypeText, "key5", "value5", 500},
	}
	cache.BatchSet(secondBatch)

	// 验证淘汰结果
	assert.Equal(t, 2, len(evictedKeys))
	stats = cache.GetStats()
	assert.Equal(t, 3, stats.Size)

	// 验证最新的条目在缓存中
	results := cache.BatchGet([]BatchGetItem{
		{model.ContentTypeVideo, "key4"},
		{model.ContentTypeText, "key5"},
	})
	assert.True(t, results["key4"].Found)
	assert.True(t, results["key5"].Found)

	// 验证最旧的条目已被淘汰
	results = cache.BatchGet([]BatchGetItem{
		{model.ContentTypeText, "key1"},
		{model.ContentTypeImage, "key2"},
	})
	assert.False(t, results["key1"].Found)
	assert.False(t, results["key2"].Found)
}

func TestBatchGetPerformance(t *testing.T) {
	cache := NewCacheManager(1000, time.Hour)

	// 准备大量测试数据
	setItems := make([]BatchSetItem, 100)
	getItems := make([]BatchGetItem, 100)
	for i := 0; i < 100; i++ {
		setItems[i] = BatchSetItem{
			ContentType: model.ContentTypeText,
			Content:     fmt.Sprintf("key%d", i),
			Value:       fmt.Sprintf("value%d", i),
			Size:        int64(100 + i),
		}
		getItems[i] = BatchGetItem{
			ContentType: model.ContentTypeText,
			Content:     fmt.Sprintf("key%d", i),
		}
	}

	// 批量设置数据
	cache.BatchSet(setItems)

	// 测试批量获取性能
	start := time.Now()
	results := cache.BatchGet(getItems)
	elapsed := time.Since(start)

	// 验证结果正确性
	assert.Equal(t, 100, len(results))
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		assert.True(t, results[key].Found)
		assert.Equal(t, fmt.Sprintf("value%d", i), results[key].Value)
	}

	// 验证性能
	assert.Less(t, elapsed.Milliseconds(), int64(100)) // 应该在100ms内完成

	// 验证统计信息
	stats := cache.GetStats()
	assert.Equal(t, 100, stats.Hits)
	assert.Equal(t, 0, stats.Misses)
	assert.True(t, stats.AvgAccessTime > 0)
}
