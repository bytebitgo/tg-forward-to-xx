package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/handlers"
	"github.com/user/tg-forward-to-xx/internal/queue"
)

var (
	configPath = flag.String("config", "config/config.yaml", "配置文件路径")
	logLevel   = flag.String("log-level", "debug", "日志级别 (debug, info, warn, error)")
	version    = "1.0.5" // 版本号
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 设置日志格式和级别（在最开始就设置）
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:       "2006-01-02 15:04:05",
		DisableLevelTruncation: true,    // 显示完整的级别名称
		PadLevelText:          true,     // 保持级别文本对齐
		DisableColors:         false,    // 启用颜色
	})

	// 设置日志级别
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logrus.Fatalf("无效的日志级别 %s: %v", *logLevel, err)
	}
	logrus.SetLevel(level)

	// 打印启动信息
	logrus.WithFields(logrus.Fields{
		"version":     version,
		"config_path": *configPath,
		"log_level":   level.String(),
	}).Info("启动服务")

	// 加载配置
	if err := config.LoadConfig(*configPath); err != nil {
		logrus.Fatalf("加载配置失败: %v", err)
	}
	
	// 打印关键配置信息
	logrus.WithFields(logrus.Fields{
		"telegram_chat_ids": config.AppConfig.Telegram.ChatIDs,
		"queue_type":       config.AppConfig.Queue.Type,
		"queue_path":       config.AppConfig.Queue.Path,
		"retry_attempts":   config.AppConfig.Retry.MaxAttempts,
		"retry_interval":   config.AppConfig.Retry.Interval,
	}).Debug("已加载配置")

	// 创建队列
	messageQueue, err := createQueue()
	if err != nil {
		logrus.Fatalf("创建消息队列失败: %v", err)
	}

	// 创建消息处理器
	handler := handlers.NewMessageHandler(messageQueue)

	// 启动处理器
	if err := handler.Start(); err != nil {
		logrus.Fatalf("启动消息处理器失败: %v", err)
	}

	// 打印指标收集状态
	if config.AppConfig.Metrics.Enabled {
		logrus.WithFields(logrus.Fields{
			"interval":     config.AppConfig.Metrics.Interval,
			"output_file": config.AppConfig.Metrics.OutputFile,
			"http_enabled": config.AppConfig.Metrics.HTTP.Enabled,
			"http_port":    config.AppConfig.Metrics.HTTP.Port,
			"http_path":    config.AppConfig.Metrics.HTTP.Path,
		}).Info("指标收集已启用")
	} else {
		logrus.Info("指标收集已禁用")
	}

	logrus.Info("服务已启动，按 Ctrl+C 停止")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logrus.Info("正在关闭服务...")
	handler.Stop()
	logrus.Info("服务已关闭")
}

// 设置日志级别
func setLogLevel(level string) {
	// 这个函数现在可以删除，因为我们在 main 函数中直接设置了日志级别
	logrus.Warnf("setLogLevel 函数已废弃，请使用命令行参数 -log-level 设置日志级别")
}

// 创建队列
func createQueue() (queue.Queue, error) {
	queueType := config.AppConfig.Queue.Type
	logrus.Infof("配置的队列类型: %s", queueType)
	
	// 检查队列路径
	if queueType == "leveldb" {
		queuePath := config.AppConfig.Queue.Path
		logrus.Infof("LevelDB 队列路径: %s", queuePath)
		
		// 检查目录是否存在
		dirPath := filepath.Dir(queuePath)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			logrus.Warnf("队列目录不存在，将尝试创建: %s", dirPath)
		}
		
		// 检查目录权限
		if info, err := os.Stat(dirPath); err == nil {
			logrus.Infof("队列目录权限: %v", info.Mode())
		}
	}

	// 初始化内存队列
	memQueue, err := queue.NewMemoryQueue()
	if err != nil {
		return nil, fmt.Errorf("初始化内存队列失败: %v", err)
	}
	logrus.Debug("内存队列初始化成功")

	// 初始化 LevelDB 队列
	_, err = queue.NewLevelDBQueue()
	if err != nil {
		logrus.Errorf("初始化 LevelDB 队列失败: %v", err)
		
		// 如果配置的是 LevelDB 队列但初始化失败，自动切换到内存队列
		if queueType == "leveldb" {
			logrus.Warnf("自动切换到内存队列作为备用方案")
			return memQueue, nil
		}
	} else {
		logrus.Debug("LevelDB 队列初始化成功")
	}

	// 创建指定类型的队列
	q, err := queue.Create(queueType)
	if err != nil {
		return nil, fmt.Errorf("创建队列失败: %v", err)
	}

	logrus.Infof("使用 %s 队列", queueType)
	return q, nil
}
