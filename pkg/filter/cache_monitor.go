package filter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 监控间隔
	Interval time.Duration

	// 日志文件路径
	LogPath string

	// 阈值告警配置
	Thresholds map[string]float64

	// 告警回调
	AlertCallback func(alert string)
}

// DefaultThresholds 默认阈值配置
var DefaultThresholds = map[string]float64{
	"hit_rate_min":        0.7,   // 最低命中率
	"memory_usage_max":    100e6, // 最大内存使用(字节)
	"avg_access_time_max": 100,   // 最大平均访问时间(毫秒)
	"expired_ratio_max":   0.2,   // 最大过期比例
}

// CacheMonitor 缓存监控服务
type CacheMonitor struct {
	cache    *CacheManager
	config   MonitorConfig
	stopChan chan struct{}
	logger   *log.Logger
}

// NewCacheMonitor 创建缓存监控服务
func NewCacheMonitor(cache *CacheManager, config MonitorConfig) (*CacheMonitor, error) {
	if config.Interval == 0 {
		config.Interval = time.Minute
	}

	if config.LogPath == "" {
		config.LogPath = "cache_stats.log"
	}

	// 创建日志目录
	logDir := filepath.Dir(config.LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// 打开日志文件
	logFile, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	monitor := &CacheMonitor{
		cache:    cache,
		config:   config,
		stopChan: make(chan struct{}),
		logger:   log.New(logFile, "", log.LstdFlags),
	}

	// 设置默认阈值
	if config.Thresholds == nil {
		config.Thresholds = DefaultThresholds
	}

	// 设置缓存阈值
	for name, value := range config.Thresholds {
		cache.SetThreshold(name, value)
	}

	// 设置阈值告警回调
	cache.SetThresholdCallback(monitor.handleThresholdAlert)

	// 设置条目淘汰回调
	cache.SetEvictionCallback(monitor.handleEviction)

	return monitor, nil
}

// Start 启动监控服务
func (m *CacheMonitor) Start() {
	go m.monitorLoop()
}

// Stop 停止监控服务
func (m *CacheMonitor) Stop() {
	close(m.stopChan)
}

// monitorLoop 监控循环
func (m *CacheMonitor) monitorLoop() {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectStats()
		case <-m.stopChan:
			return
		}
	}
}

// collectStats 收集统计信息
func (m *CacheMonitor) collectStats() {
	stats := m.cache.GetStats()

	// 记录统计信息
	statsJSON, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		m.logger.Printf("Error marshaling stats: %v", err)
		return
	}

	m.logger.Printf("Cache Stats:\n%s\n", string(statsJSON))

	// 检查关键指标
	m.checkMetrics(stats)
}

// checkMetrics 检查关键指标
func (m *CacheMonitor) checkMetrics(stats CacheStats) {
	// 检查内存使用
	memoryUsageMB := float64(stats.MemoryUsage) / (1024 * 1024)
	if memoryUsageMB > m.config.Thresholds["memory_usage_max"]/(1024*1024) {
		m.alert(fmt.Sprintf("High memory usage: %.2f MB (threshold: %.2f MB)",
			memoryUsageMB, m.config.Thresholds["memory_usage_max"]/(1024*1024)))
	}

	// 检查命中率
	if stats.HitRate < m.config.Thresholds["hit_rate_min"] {
		m.alert(fmt.Sprintf("Low hit rate: %.2f%% (threshold: %.2f%%)",
			stats.HitRate*100, m.config.Thresholds["hit_rate_min"]*100))
	}

	// 检查响应时间
	if stats.AvgAccessTime > m.config.Thresholds["avg_access_time_max"] {
		m.alert(fmt.Sprintf("High average access time: %.2f ms (threshold: %.2f ms)",
			stats.AvgAccessTime, m.config.Thresholds["avg_access_time_max"]))
	}

	// 检查过期条目比例
	expiredRatio := float64(stats.ExpiredEntries) / float64(stats.Size)
	if expiredRatio > m.config.Thresholds["expired_ratio_max"] {
		m.alert(fmt.Sprintf("High expired entries ratio: %.2f%% (threshold: %.2f%%)",
			expiredRatio*100, m.config.Thresholds["expired_ratio_max"]*100))
	}
}

// handleThresholdAlert 处理阈值告警
func (m *CacheMonitor) handleThresholdAlert(stats CacheStats) {
	alert := "Cache threshold alert:\n"

	if stats.HitRate < m.config.Thresholds["hit_rate_min"] {
		alert += fmt.Sprintf("- Low hit rate: %.2f%% (threshold: %.2f%%)\n",
			stats.HitRate*100, m.config.Thresholds["hit_rate_min"]*100)
	}

	if stats.MemoryUsage > int64(m.config.Thresholds["memory_usage_max"]) {
		alert += fmt.Sprintf("- High memory usage: %d bytes (threshold: %.0f bytes)\n",
			stats.MemoryUsage, m.config.Thresholds["memory_usage_max"])
	}

	if stats.AvgAccessTime > m.config.Thresholds["avg_access_time_max"] {
		alert += fmt.Sprintf("- High average access time: %.2f ms (threshold: %.2f ms)\n",
			stats.AvgAccessTime, m.config.Thresholds["avg_access_time_max"])
	}

	m.alert(alert)
}

// handleEviction 处理缓存条目淘汰
func (m *CacheMonitor) handleEviction(key string, entry CacheEntry) {
	m.logger.Printf("Cache entry evicted: key=%s, age=%.2fs, accesses=%d, size=%d bytes\n",
		key, time.Since(entry.Timestamp).Seconds(), entry.AccessCount, entry.Size)
}

// alert 发送告警
func (m *CacheMonitor) alert(message string) {
	m.logger.Printf("ALERT: %s", message)

	if m.config.AlertCallback != nil {
		m.config.AlertCallback(message)
	}
}
