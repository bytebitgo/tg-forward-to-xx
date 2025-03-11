package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/api"
	"github.com/user/tg-forward-to-xx/internal/config"
	"github.com/user/tg-forward-to-xx/internal/handlers"
	"github.com/user/tg-forward-to-xx/internal/queue"
	"github.com/user/tg-forward-to-xx/internal/storage"
	"golang.org/x/sys/unix"
)

var (
	// Version 程序版本号
	Version = "1.0.20"
	configPath  string
	showVersion bool
	logLevel    string
	metricsPort int
	httpPort    int
)

func init() {
	flag.StringVar(&configPath, "config", config.GetConfigPath(), "配置文件路径")
	flag.StringVar(&logLevel, "log-level", "info", "日志级别")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.IntVar(&httpPort, "http-port", 8080, "HTTP API 端口")
	flag.IntVar(&metricsPort, "metrics-port", 9090, "指标服务端口")
}

// fileHook 用于将日志同时写入文件
type fileHook struct {
	logger *logrus.Logger
}

// Fire 实现 logrus.Hook 接口
func (h *fileHook) Fire(entry *logrus.Entry) error {
	// 使用 WithFields 来保持所有字段的一致性
	logEntry := h.logger.WithFields(entry.Data)
	
	switch entry.Level {
	case logrus.PanicLevel:
		logEntry.Panic(entry.Message)
	case logrus.FatalLevel:
		logEntry.Fatal(entry.Message)
	case logrus.ErrorLevel:
		logEntry.Error(entry.Message)
	case logrus.WarnLevel:
		logEntry.Warn(entry.Message)
	case logrus.InfoLevel:
		logEntry.Info(entry.Message)
	case logrus.DebugLevel:
		logEntry.Debug(entry.Message)
	case logrus.TraceLevel:
		logEntry.Trace(entry.Message)
	}
	
	return nil
}

// Levels 实现 logrus.Hook 接口
func (h *fileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func main() {
	flag.Parse()

	// 加载配置
	if err := config.LoadConfig(configPath); err != nil {
		logrus.Fatalf("加载配置失败: %v", err)
	}

	// 设置日志格式
	formatter := &logrus.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:       "2006-01-02 15:04:05",
		DisableLevelTruncation: true,    // 显示完整的级别名称
		PadLevelText:          true,     // 保持级别文本对齐
		DisableColors:         false,    // 启用颜色
		ForceColors:          true,     // 强制启用颜色，即使不是终端
	}
	logrus.SetFormatter(formatter)

	// 设置日志级别
	var level logrus.Level
	var err error
	
	// 优先使用命令行参数的日志级别
	if logLevel != "" {
		level, err = logrus.ParseLevel(logLevel)
	} else if config.AppConfig.Log.Level != "" {
		// 如果命令行参数未指定，使用配置文件中的日志级别
		level, err = logrus.ParseLevel(config.AppConfig.Log.Level)
	} else {
		// 默认使用 info 级别
		level = logrus.InfoLevel
	}

	if err != nil {
		logrus.Fatalf("无效的日志级别: %v", err)
	}
	logrus.SetLevel(level)

	// 配置日志输出
	if config.AppConfig.Log.FilePath != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(config.AppConfig.Log.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logrus.Fatalf("创建日志目录失败: %v", err)
		}

		// 打开日志文件
		logFile, err := os.OpenFile(config.AppConfig.Log.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logrus.Fatalf("打开日志文件失败: %v", err)
		}

		// 创建多输出写入器
		mw := io.MultiWriter(os.Stdout, logFile)
		logrus.SetOutput(mw)
	} else {
		// 如果未配置日志文件路径，使用默认路径
		defaultLogPath := "/var/log/tg-forward/main.log"
		logDir := filepath.Dir(defaultLogPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logrus.Fatalf("创建默认日志目录失败: %v", err)
		}

		// 打开默认日志文件
		logFile, err := os.OpenFile(defaultLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logrus.Fatalf("打开默认日志文件失败: %v", err)
		}

		// 创建多输出写入器
		mw := io.MultiWriter(os.Stdout, logFile)
		logrus.SetOutput(mw)
	}

	// 显示版本信息
	if showVersion {
		fmt.Printf("tg-forward 版本 %s\n", Version)
		os.Exit(0)
	}

	// 打印启动信息
	logrus.WithFields(logrus.Fields{
		"version":     Version,
		"config_path": configPath,
		"log_level":   level.String(),
		"log_file":    config.AppConfig.Log.FilePath,
		"pid":        os.Getpid(),
	}).Info("🚀 启动 Telegram 转发服务")

	// 打印关键配置信息
	logrus.WithFields(logrus.Fields{
		"telegram_chat_ids": config.AppConfig.Telegram.ChatIDs,
		"queue_type":       config.AppConfig.Queue.Type,
		"queue_path":       config.AppConfig.Queue.Path,
		"retry_attempts":   config.AppConfig.Retry.MaxAttempts,
		"retry_interval":   config.AppConfig.Retry.Interval,
	}).Debug("已加载配置")

	// 初始化聊天记录存储
	chatHistoryStorage, err := storage.NewChatHistoryStorage()
	if err != nil {
		logrus.Fatalf("初始化聊天记录存储失败: %v", err)
	}
	defer chatHistoryStorage.Close()

	// 创建消息队列
	messageQueue, err := createQueue()
	if err != nil {
		logrus.Fatalf("创建消息队列失败: %v", err)
	}

	// 创建消息处理器
	messageHandler, err := handlers.NewMessageHandler(messageQueue, chatHistoryStorage)
	if err != nil {
		logrus.Fatalf("创建消息处理器失败: %v", err)
	}

	// 启动消息处理
	if err := messageHandler.Start(); err != nil {
		logrus.Fatalf("启动消息处理失败: %v", err)
	}

	// 创建 API 处理器
	chatHistoryHandler := api.NewChatHistoryHandler(chatHistoryStorage)

	// 设置 HTTP 路由
	http.HandleFunc("/api/chat/history", chatHistoryHandler.QueryHandler)
	http.HandleFunc("/api/chat/history/user", chatHistoryHandler.QueryByUserHandler)
	http.HandleFunc("/api/chat/history/export", chatHistoryHandler.ExportHandler)

	// 启动 HTTP 服务
	go func() {
		addr := fmt.Sprintf(":%d", httpPort)
		logrus.Infof("HTTP API 服务启动在 %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			logrus.Errorf("HTTP 服务启动失败: %v", err)
		}
	}()

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
	messageHandler.Stop()
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
		
		// 检查并创建完整的队列目录
		if _, err := os.Stat(queuePath); os.IsNotExist(err) {
			logrus.Infof("队列目录不存在，正在创建: %s", queuePath)
			if err := os.MkdirAll(queuePath, 0755); err != nil {
				logrus.Errorf("创建队列目录失败: %v", err)
				return nil, fmt.Errorf("创建队列目录失败: %v", err)
			}
			logrus.Info("队列目录创建成功")
		}
		
		// 检查目录权限
		if info, err := os.Stat(queuePath); err == nil {
			mode := info.Mode()
			logrus.Infof("队列目录权限: %v", mode)
			
			// 检查目录是否可写
			if err := unix.Access(queuePath, unix.W_OK); err != nil {
				logrus.Errorf("队列目录不可写: %v", err)
				return nil, fmt.Errorf("队列目录不可写: %v", err)
			}
			logrus.Info("队列目录权限检查通过")
		} else {
			logrus.Errorf("检查队列目录状态失败: %v", err)
			return nil, fmt.Errorf("检查队列目录状态失败: %v", err)
		}
	}

	// 尝试创建 LevelDB 队列
	logrus.Debug("开始创建队列")
	leveldbQueue, err := queue.Create(queueType)
	if err != nil {
		logrus.Errorf("创建 LevelDB 队列失败: %v", err)
		
		// 初始化内存队列作为备用
		logrus.Info("尝试初始化内存队列作为备用")
		memQueue, memErr := queue.NewMemoryQueue()
		if memErr != nil {
			logrus.Errorf("初始化内存队列失败: %v", memErr)
			return nil, fmt.Errorf("初始化内存队列失败: %v", memErr)
		}
		
		logrus.Warn("自动切换到内存队列作为备用方案")
		return memQueue, nil
	}

	logrus.Info("成功创建 LevelDB 队列")
	return leveldbQueue, nil
}
