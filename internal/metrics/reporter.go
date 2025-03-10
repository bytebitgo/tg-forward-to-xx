package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/queue"
)

// Reporter 指标报告器
type Reporter struct {
	queue       queue.Queue
	stopChan    chan struct{}
	interval    time.Duration
	metricsFile string
	httpServer  *HTTPServer
}

// NewReporter 创建新的指标报告器
func NewReporter(q queue.Queue, interval time.Duration, metricsFile string) *Reporter {
	reporter := &Reporter{
		queue:       q,
		stopChan:    make(chan struct{}),
		interval:    interval,
		metricsFile: metricsFile,
	}

	// 如果启用了 HTTP 服务，创建 HTTP 服务器
	if config.AppConfig.Metrics.HTTP.Enabled {
		reporter.httpServer = NewHTTPServer(
			config.AppConfig.Metrics.HTTP.Port,
			config.AppConfig.Metrics.HTTP.Path,
		)
	}

	return reporter
}

// Start 启动指标报告器
func (r *Reporter) Start() {
	// 启动指标收集协程
	go r.run()

	// 如果配置了 HTTP 服务，启动 HTTP 服务
	if r.httpServer != nil {
		if err := r.httpServer.Start(); err != nil {
			logrus.Errorf("启动 HTTP 服务失败: %v", err)
		}
	}
}

// Stop 停止指标报告器
func (r *Reporter) Stop() {
	close(r.stopChan)

	// 如果启动了 HTTP 服务，停止 HTTP 服务
	if r.httpServer != nil {
		if err := r.httpServer.Stop(); err != nil {
			logrus.Errorf("停止 HTTP 服务失败: %v", err)
		}
	}
}

// 运行指标报告循环
func (r *Reporter) run() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// 立即收集一次指标
	r.collectAndReport()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.collectAndReport()
		}
	}
}

// 收集并报告指标
func (r *Reporter) collectAndReport() {
	// 获取队列大小
	size, err := r.queue.Size()
	if err != nil {
		logrus.Errorf("获取队列大小失败: %v", err)
		return
	}

	// 更新指标
	SetQueueSize(size)

	// 获取所有指标
	metrics := GetMetrics()

	// 输出到日志
	r.logMetrics(metrics)

	// 如果配置了指标文件，则写入文件
	if r.metricsFile != "" {
		r.writeMetricsToFile(metrics)
	}
}

// 记录指标到日志
func (r *Reporter) logMetrics(metrics map[string]interface{}) {
	logrus.WithFields(logrus.Fields{
		"queue_size":         metrics["queue_size"],
		"processed_messages": metrics["processed_messages"],
		"failed_messages":    metrics["failed_messages"],
		"retry_messages":     metrics["retry_messages"],
		"last_update_time":   metrics["last_update_time"],
	}).Info("队列统计信息")
}

// 将指标写入文件
func (r *Reporter) writeMetricsToFile(metrics map[string]interface{}) {
	if r.metricsFile == "" {
		return
	}

	// 确保目录存在
	dir := filepath.Dir(r.metricsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logrus.Errorf("创建指标文件目录失败: %v", err)
		return
	}

	// 序列化指标数据
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		logrus.Errorf("序列化指标数据失败: %v", err)
		return
	}

	// 写入文件
	if err := os.WriteFile(r.metricsFile, data, 0644); err != nil {
		logrus.Errorf("写入指标文件失败: %v", err)
		return
	}

	logrus.Debugf("指标数据已写入文件: %s", r.metricsFile)
}
