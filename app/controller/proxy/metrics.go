package proxy

import (
	"sync"
	"sync/atomic"
	"time"
)

// RequestMetrics 请求性能指标
type RequestMetrics struct {
	TotalRequests   int64         `json:"totalRequests"`
	SuccessRequests int64         `json:"successRequests"`
	FailedRequests  int64         `json:"failedRequests"`
	TotalDuration   time.Duration `json:"totalDuration"`
	AvgDuration     time.Duration `json:"avgDuration"`
	MaxDuration     time.Duration `json:"maxDuration"`
	MinDuration     time.Duration `json:"minDuration"`
	mutex           sync.RWMutex
}

// NewRequestMetrics 创建新的请求指标
func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{
		MinDuration: time.Hour, // 初始化为很大的值，方便后续比较
	}
}

// RecordRequest 记录一次请求
func (m *RequestMetrics) RecordRequest(duration time.Duration, success bool) {
	atomic.AddInt64(&m.TotalRequests, 1)
	
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalDuration += duration

	if success {
		atomic.AddInt64(&m.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&m.FailedRequests, 1)
	}

	if duration > m.MaxDuration {
		m.MaxDuration = duration
	}
	if duration < m.MinDuration || m.MinDuration == time.Hour {
		m.MinDuration = duration
	}

	total := atomic.LoadInt64(&m.TotalRequests)
	if total > 0 {
		m.AvgDuration = m.TotalDuration / time.Duration(total)
	}
}

// GetStats 获取统计信息
func (m *RequestMetrics) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	total := atomic.LoadInt64(&m.TotalRequests)
	success := atomic.LoadInt64(&m.SuccessRequests)
	failed := atomic.LoadInt64(&m.FailedRequests)

	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	return map[string]interface{}{
		"totalRequests":   total,
		"successRequests": success,
		"failedRequests":  failed,
		"successRate":     successRate,
		"avgDurationMs":   m.AvgDuration.Milliseconds(),
		"maxDurationMs":   m.MaxDuration.Milliseconds(),
		"minDurationMs":   m.MinDuration.Milliseconds(),
	}
}

// Reset 重置指标
func (m *RequestMetrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	atomic.StoreInt64(&m.TotalRequests, 0)
	atomic.StoreInt64(&m.SuccessRequests, 0)
	atomic.StoreInt64(&m.FailedRequests, 0)
	m.TotalDuration = 0
	m.AvgDuration = 0
	m.MaxDuration = 0
	m.MinDuration = time.Hour
}
