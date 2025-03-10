package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/handlers"
	"github.com/user/tg-forward-to-xx/internal/queue"
)

var (
	configPath = flag.String("config", "config/config.yaml", "配置文件路径")
	logLevel   = flag.String("log-level", "info", "日志级别 (debug, info, warn, error)")
	version    = "1.0.4" // 版本号
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 设置日志级别
	setLogLevel(*logLevel)

	// 打印版本信息
	logrus.Infof("Telegram 转发到钉钉 v%s", version)
	logrus.Infof("配置文件: %s", *configPath)

	// 加载配置
	if err := config.LoadConfig(*configPath); err != nil {
		logrus.Fatalf("加载配置失败: %v", err)
	}

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
		logrus.Infof("指标收集已启用，间隔: %d秒，输出文件: %s",
			config.AppConfig.Metrics.Interval,
			config.AppConfig.Metrics.OutputFile)

		// 打印 HTTP 服务状态
		if config.AppConfig.Metrics.HTTP.Enabled {
			logrus.Infof("指标 HTTP 服务已启用，端口: %d，路径: %s",
				config.AppConfig.Metrics.HTTP.Port,
				config.AppConfig.Metrics.HTTP.Path)
		} else {
			logrus.Info("指标 HTTP 服务已禁用")
		}
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
	switch level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	// 设置日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// 创建队列
func createQueue() (queue.Queue, error) {
	queueType := config.AppConfig.Queue.Type

	// 初始化内存队列
	if _, err := queue.NewMemoryQueue(); err != nil {
		return nil, fmt.Errorf("初始化内存队列失败: %v", err)
	}

	// 初始化 LevelDB 队列
	if _, err := queue.NewLevelDBQueue(); err != nil {
		return nil, fmt.Errorf("初始化 LevelDB 队列失败: %v", err)
	}

	// 创建指定类型的队列
	q, err := queue.Create(queueType)
	if err != nil {
		return nil, fmt.Errorf("创建队列失败: %v", err)
	}

	logrus.Infof("使用 %s 队列", queueType)
	return q, nil
}
