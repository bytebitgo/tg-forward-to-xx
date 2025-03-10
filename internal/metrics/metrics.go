package metrics

import (
	"sync"
	"time"
)

// QueueMetrics 队列指标结构
type QueueMetrics struct {
	mu                sync.RWMutex
	QueueSize         int       // 队列大小
	ProcessedMessages int64     // 已处理消息数
	FailedMessages    int64     // 失败消息数
	RetryMessages     int64     // 重试消息数
	LastUpdateTime    time.Time // 最后更新时间
	StartTime         time.Time // 启动时间

	// 新增指标
	TotalProcessingTime time.Duration   // 总处理时间
	MessageLatencies    []time.Duration // 最近100条消息的处理延迟
	LastMinuteMessages  int64           // 最近一分钟处理的消息数
	LastMinuteTime      time.Time       // 最近一分钟的开始时间
	TotalRetryCount     int64           // 总重试次数
}

var (
	// DefaultMetrics 默认指标实例
	DefaultMetrics = NewQueueMetrics()
)

// NewQueueMetrics 创建新的队列指标实例
func NewQueueMetrics() *QueueMetrics {
	now := time.Now()
	return &QueueMetrics{
		LastUpdateTime:   now,
		StartTime:        now,
		LastMinuteTime:   now,
		MessageLatencies: make([]time.Duration, 0, 100),
	}
}

// SetQueueSize 设置队列大小
func (m *QueueMetrics) SetQueueSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueueSize = size
	m.LastUpdateTime = time.Now()
}

// GetQueueSize 获取队列大小
func (m *QueueMetrics) GetQueueSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueueSize
}

// IncrementProcessedMessages 增加已处理消息计数
func (m *QueueMetrics) IncrementProcessedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProcessedMessages++
	m.LastUpdateTime = time.Now()
}

// IncrementFailedMessages 增加失败消息计数
func (m *QueueMetrics) IncrementFailedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedMessages++
	m.LastUpdateTime = time.Now()
}

// IncrementRetryMessages 增加重试消息计数
func (m *QueueMetrics) IncrementRetryMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RetryMessages++
	m.LastUpdateTime = time.Now()
}

// AddMessageLatency 添加消息处理延迟
func (m *QueueMetrics) AddMessageLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalProcessingTime += latency

	// 保持最近100条消息的延迟记录
	if len(m.MessageLatencies) >= 100 {
		m.MessageLatencies = m.MessageLatencies[1:]
	}
	m.MessageLatencies = append(m.MessageLatencies, latency)

	// 更新每分钟处理消息数
	now := time.Now()
	if now.Sub(m.LastMinuteTime) >= time.Minute {
		m.LastMinuteMessages = 1
		m.LastMinuteTime = now
	} else {
		m.LastMinuteMessages++
	}
}

// GetAverageLatency 获取平均处理延迟
func (m *QueueMetrics) GetAverageLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ProcessedMessages == 0 {
		return 0
	}
	return m.TotalProcessingTime / time.Duration(m.ProcessedMessages)
}

// GetThroughput 获取吞吐量（每分钟处理消息数）
func (m *QueueMetrics) GetThroughput() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return float64(m.LastMinuteMessages)
}

// GetSuccessRate 获取消息处理成功率
func (m *QueueMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.ProcessedMessages + m.FailedMessages
	if total == 0 {
		return 100.0
	}
	return float64(m.ProcessedMessages) / float64(total) * 100
}

// GetAverageRetryCount 获取平均重试次数
func (m *QueueMetrics) GetAverageRetryCount() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ProcessedMessages == 0 {
		return 0
	}
	return float64(m.TotalRetryCount) / float64(m.ProcessedMessages)
}

// GetQueuePressure 获取队列积压程度
func (m *QueueMetrics) GetQueuePressure() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.LastMinuteMessages == 0 {
		return float64(m.QueueSize)
	}
	return float64(m.QueueSize) / float64(m.LastMinuteMessages)
}

// IncrementRetryCount 增加重试计数
func (m *QueueMetrics) IncrementRetryCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRetryCount++
}

// GetMetrics 获取所有指标
func (m *QueueMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := m.GetAverageLatency()
	var p95Latency time.Duration
	if len(m.MessageLatencies) > 0 {
		idx := int(float64(len(m.MessageLatencies)) * 0.95)
		if idx >= len(m.MessageLatencies) {
			idx = len(m.MessageLatencies) - 1
		}
		p95Latency = m.MessageLatencies[idx]
	}

	return map[string]interface{}{
		"queue_size":         m.QueueSize,
		"processed_messages": m.ProcessedMessages,
		"failed_messages":    m.FailedMessages,
		"retry_messages":     m.RetryMessages,
		"last_update_time":   m.LastUpdateTime.Format(time.RFC3339),
		"uptime_seconds":     time.Since(m.StartTime).Seconds(),
		"avg_latency_ms":     avgLatency.Milliseconds(),
		"p95_latency_ms":     p95Latency.Milliseconds(),
		"throughput_per_min": m.GetThroughput(),
		"success_rate":       m.GetSuccessRate(),
		"avg_retry_count":    m.GetAverageRetryCount(),
		"queue_pressure":     m.GetQueuePressure(),
		"total_retry_count":  m.TotalRetryCount,
	}
}

// ResetCounters 重置计数器
func (m *QueueMetrics) ResetCounters() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProcessedMessages = 0
	m.FailedMessages = 0
	m.RetryMessages = 0
	m.LastUpdateTime = time.Now()
}

// SetQueueSize 设置全局队列大小
func SetQueueSize(size int) {
	DefaultMetrics.SetQueueSize(size)
}

// IncrementProcessedMessages 增加全局已处理消息计数
func IncrementProcessedMessages() {
	DefaultMetrics.IncrementProcessedMessages()
}

// IncrementFailedMessages 增加全局失败消息计数
func IncrementFailedMessages() {
	DefaultMetrics.IncrementFailedMessages()
}

// IncrementRetryMessages 增加全局重试消息计数
func IncrementRetryMessages() {
	DefaultMetrics.IncrementRetryMessages()
}

// GetMetrics 获取全局指标
func GetMetrics() map[string]interface{} {
	return DefaultMetrics.GetMetrics()
}

// ResetCounters 重置全局计数器
func ResetCounters() {
	DefaultMetrics.ResetCounters()
}

// AddMessageLatency 添加全局消息处理延迟
func AddMessageLatency(latency time.Duration) {
	DefaultMetrics.AddMessageLatency(latency)
}

// IncrementRetryCount 增加全局重试计数
func IncrementRetryCount() {
	DefaultMetrics.IncrementRetryCount()
}
