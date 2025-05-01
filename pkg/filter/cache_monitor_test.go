package filter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BinLe1988/multi-agent-chatter/pkg/filter/model"
	"github.com/stretchr/testify/assert"
)

func TestCacheMonitor(t *testing.T) {
	// 创建临时日志目录
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_cache.log")

	// 创建缓存管理器
	cache := NewCacheManager(10, time.Second)

	// 创建监控配置
	config := MonitorConfig{
		Interval: 100 * time.Millisecond,
		LogPath:  logPath,
		Thresholds: map[string]float64{
			"hit_rate_min":        0.8,  // 更严格的命中率要求
			"memory_usage_max":    50e6, // 更小的内存限制
			"avg_access_time_max": 50,   // 更短的响应时间要求
			"expired_ratio_max":   0.1,  // 更低的过期比例
		},
	}

	// 记录告警信息
	var alerts []string
	config.AlertCallback = func(alert string) {
		alerts = append(alerts, alert)
	}

	// 创建监控服务
	monitor, err := NewCacheMonitor(cache, config)
	assert.NoError(t, err)
	assert.NotNil(t, monitor)

	// 启动监控
	monitor.Start()
	defer monitor.Stop()

	// 添加测试数据
	testData := []struct {
		contentType model.ContentType
		content     string
		value       interface{}
		size        int64
	}{
		{model.ContentTypeText, "test1", "value1", 100},
		{model.ContentTypeImage, "test2", "value2", 200},
		{model.ContentTypeAudio, "test3", "value3", 300},
		{model.ContentTypeVideo, "test4", "value4", 400},
	}

	for _, data := range testData {
		cache.Set(data.contentType, data.content, data.value, data.size)
	}

	// 模拟缓存访问
	for i := 0; i < 5; i++ {
		for _, data := range testData {
			value, ok := cache.Get(data.contentType, data.content)
			assert.True(t, ok)
			assert.Equal(t, data.value, value)
		}
	}

	// 等待监控数据收集
	time.Sleep(200 * time.Millisecond)

	// 验证日志文件
	logContent, err := os.ReadFile(logPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, logContent)

	// 解析日志中的统计信息
	var stats CacheStats
	lines := string(logContent)
	assert.Contains(t, lines, "Cache Stats")

	// 验证统计信息
	assert.Equal(t, len(testData), stats.Size)
	assert.True(t, stats.HitRate > 0.8)
	assert.True(t, stats.MemoryUsage > 0)
	assert.True(t, stats.AvgAccessTime >= 0)

	// 验证按类型统计
	for _, data := range testData {
		typeStats, ok := stats.TypeStats[data.contentType]
		assert.True(t, ok)
		assert.Equal(t, 1, typeStats.Count)
		assert.True(t, typeStats.HitRate > 0)
		assert.True(t, typeStats.AvgLatency >= 0)
	}

	// 测试过期清理
	time.Sleep(time.Second + 100*time.Millisecond)

	// 再次获取数据，应该已过期
	for _, data := range testData {
		value, ok := cache.Get(data.contentType, data.content)
		assert.False(t, ok)
		assert.Nil(t, value)
	}

	// 等待监控数据更新
	time.Sleep(200 * time.Millisecond)

	// 验证是否有告警
	assert.NotEmpty(t, alerts)
	for _, alert := range alerts {
		t.Logf("Received alert: %s", alert)
	}
}

func TestCacheMonitorThresholds(t *testing.T) {
	// 创建临时日志目录
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_thresholds.log")

	// 创建缓存管理器
	cache := NewCacheManager(5, time.Second)

	// 创建监控配置，设置较低的阈值以触发告警
	config := MonitorConfig{
		Interval: 100 * time.Millisecond,
		LogPath:  logPath,
		Thresholds: map[string]float64{
			"hit_rate_min":        0.9,  // 非常高的命中率要求
			"memory_usage_max":    1000, // 非常低的内存限制
			"avg_access_time_max": 10,   // 非常短的响应时间要求
			"expired_ratio_max":   0.05, // 非常低的过期比例
		},
	}

	// 记录告警信息
	var alerts []string
	config.AlertCallback = func(alert string) {
		alerts = append(alerts, alert)
	}

	// 创建监控服务
	monitor, err := NewCacheMonitor(cache, config)
	assert.NoError(t, err)

	// 启动监控
	monitor.Start()
	defer monitor.Stop()

	// 添加大量数据以触发阈值告警
	for i := 0; i < 10; i++ {
		cache.Set(model.ContentTypeText,
			fmt.Sprintf("test%d", i),
			fmt.Sprintf("value%d", i),
			1000)
	}

	// 模拟一些缓存未命中
	for i := 10; i < 15; i++ {
		_, _ = cache.Get(model.ContentTypeText, fmt.Sprintf("test%d", i))
	}

	// 等待监控检查
	time.Sleep(200 * time.Millisecond)

	// 验证是否触发了告警
	assert.NotEmpty(t, alerts)

	// 验证告警内容
	var foundMemoryAlert, foundHitRateAlert bool
	for _, alert := range alerts {
		if strings.Contains(alert, "High memory usage") {
			foundMemoryAlert = true
		}
		if strings.Contains(alert, "Low hit rate") {
			foundHitRateAlert = true
		}
	}

	assert.True(t, foundMemoryAlert, "Should have memory usage alert")
	assert.True(t, foundHitRateAlert, "Should have hit rate alert")
}

func TestCacheMonitorEviction(t *testing.T) {
	// 创建临时日志目录
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_eviction.log")

	// 创建小容量的缓存管理器
	cache := NewCacheManager(3, time.Second)

	// 创建监控配置
	config := MonitorConfig{
		Interval: 100 * time.Millisecond,
		LogPath:  logPath,
	}

	// 记录淘汰的条目
	var evicted []string
	config.AlertCallback = func(alert string) {
		if strings.Contains(alert, "Cache entry evicted") {
			parts := strings.Split(alert, "key=")
			if len(parts) > 1 {
				key := strings.Split(parts[1], ",")[0]
				evicted = append(evicted, key)
			}
		}
	}

	// 创建监控服务
	monitor, err := NewCacheMonitor(cache, config)
	assert.NoError(t, err)

	// 启动监控
	monitor.Start()
	defer monitor.Stop()

	// 添加超过容量的数据以触发淘汰
	for i := 0; i < 5; i++ {
		cache.Set(model.ContentTypeText,
			fmt.Sprintf("test%d", i),
			fmt.Sprintf("value%d", i),
			100)
		time.Sleep(10 * time.Millisecond)
	}

	// 等待监控检查
	time.Sleep(200 * time.Millisecond)

	// 验证是否有条目被淘汰
	assert.NotEmpty(t, evicted)
	assert.Equal(t, 2, len(evicted)) // 应该有2个条目被淘汰

	// 验证缓存大小
	stats := cache.GetStats()
	assert.Equal(t, 3, stats.Size)
}

func TestCacheMonitorCustomConfig(t *testing.T) {
	// 创建临时日志目录
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_custom.log")

	// 创建缓存管理器
	cache := NewCacheManager(10, time.Second)

	// 使用默认配置创建监控服务
	monitor1, err := NewCacheMonitor(cache, MonitorConfig{
		LogPath: logPath,
	})
	assert.NoError(t, err)
	assert.Equal(t, time.Minute, monitor1.config.Interval)
	assert.NotNil(t, monitor1.config.Thresholds)

	// 使用自定义配置创建监控服务
	customConfig := MonitorConfig{
		Interval: 30 * time.Second,
		LogPath:  logPath,
		Thresholds: map[string]float64{
			"custom_threshold": 123.45,
		},
	}

	monitor2, err := NewCacheMonitor(cache, customConfig)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, monitor2.config.Interval)
	assert.Equal(t, 123.45, monitor2.config.Thresholds["custom_threshold"])
}
