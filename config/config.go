package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigPath 默认配置文件路径
	DefaultConfigPath = "/etc/tg-forward/config.yaml"
)

//del unused
// Config 应用程序配置
type Config struct {
	Telegram struct {
		Token   string  `mapstructure:"token"`
		ChatIDs []int64 `mapstructure:"chat_ids"`
	} `mapstructure:"telegram"`

	DingTalk struct {
		WebhookURL string `mapstructure:"webhook_url"`
		Secret     string `mapstructure:"secret"`
	} `mapstructure:"dingtalk"`

	Log struct {
		Level    string `mapstructure:"level"`     // 日志级别
		FilePath string `mapstructure:"file_path"` // 日志文件路径
		MaxSize  int    `mapstructure:"max_size"`  // 单个日志文件最大大小（MB）
		MaxFiles int    `mapstructure:"max_files"` // 最大保留文件数
	} `mapstructure:"log"`

	Queue struct {
		Type string `mapstructure:"type"` // "memory" 或 "leveldb"
		Path string `mapstructure:"path"` // LevelDB 存储路径
	} `mapstructure:"queue"`

	Retry struct {
		MaxAttempts int   `mapstructure:"max_attempts"`
		Interval    int64 `mapstructure:"interval"` // 重试间隔（秒）
	} `mapstructure:"retry"`

	Metrics struct {
		Enabled    bool   `mapstructure:"enabled"`     // 是否启用指标收集
		Interval   int    `mapstructure:"interval"`    // 收集间隔（秒）
		OutputFile string `mapstructure:"output_file"` // 指标输出文件路径
		HTTP       struct {
			Enabled    bool   `mapstructure:"enabled"`     // 是否启用 HTTP 服务
			Port       int    `mapstructure:"port"`        // HTTP 服务端口
			Path       string `mapstructure:"path"`        // 指标 API 路径
			Auth       bool   `mapstructure:"auth"`        // 是否启用认证
			APIKey     string `mapstructure:"api_key"`     // API Key
			HeaderName string `mapstructure:"header_name"` // API Key 请求头名称
			TLS        struct {
				Enabled    bool   `mapstructure:"enabled"`     // 是否启用 HTTPS
				CertFile   string `mapstructure:"cert_file"`   // 证书文件路径
				KeyFile    string `mapstructure:"key_file"`    // 私钥文件路径
				Port       int    `mapstructure:"port"`        // HTTPS 端口（可选，默认与 HTTP 端口相同）
				ForceHTTPS bool   `mapstructure:"force_https"` // 是否强制使用 HTTPS
			} `mapstructure:"tls"`
		} `mapstructure:"http"`
	} `mapstructure:"metrics"`

	S3 struct {
		Endpoint        string `mapstructure:"endpoint"`
		Region          string `mapstructure:"region"`
		Bucket          string `mapstructure:"bucket"`
		AccessKeyID     string `mapstructure:"access_key_id"`
		SecretAccessKey string `mapstructure:"secret_access_key"`
		UseSSL         bool   `mapstructure:"use_ssl"`
		PublicBaseURL  string `mapstructure:"public_base_url"`
	} `mapstructure:"s3"`
}

var (
	// AppConfig 全局配置实例
	AppConfig Config
)

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 1. 检查环境变量
	if envPath := os.Getenv("TG_FORWARD_CONFIG"); envPath != "" {
		return envPath
	}

	// 2. 检查当前目录
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// 3. 检查默认路径
	if _, err := os.Stat(DefaultConfigPath); err == nil {
		return DefaultConfigPath
	}

	// 4. 返回默认路径（即使不存在）
	return DefaultConfigPath
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	logrus.WithField("path", configPath).Info("正在加载配置文件")

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证必要的配置项
	if err := validateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"telegram_token":     maskToken(AppConfig.Telegram.Token),
		"telegram_chat_ids": AppConfig.Telegram.ChatIDs,
		"dingtalk_webhook": maskURL(AppConfig.DingTalk.WebhookURL),
		"dingtalk_secret": maskToken(AppConfig.DingTalk.Secret),
		"queue_type":      AppConfig.Queue.Type,
		"queue_path":      AppConfig.Queue.Path,
	}).Info("配置加载完成")

	return nil
}

// validateConfig 验证配置是否完整
func validateConfig() error {
	if AppConfig.Telegram.Token == "" {
		return fmt.Errorf("Telegram Bot Token 未配置")
	}

	if len(AppConfig.Telegram.ChatIDs) == 0 {
		logrus.Warn("未配置任何 Telegram 聊天 ID，机器人将不会转发任何消息")
	}

	if AppConfig.DingTalk.WebhookURL == "" {
		return fmt.Errorf("钉钉 Webhook URL 未配置")
	}

	// 验证日志配置
	if AppConfig.Log.Level != "" {
		if _, err := logrus.ParseLevel(AppConfig.Log.Level); err != nil {
			return fmt.Errorf("无效的日志级别 %s: %v", AppConfig.Log.Level, err)
		}
	}

	if AppConfig.Log.FilePath != "" {
		// 检查日志目录是否可写
		logDir := filepath.Dir(AppConfig.Log.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %v", err)
		}
		if info, err := os.Stat(logDir); err != nil {
			return fmt.Errorf("检查日志目录失败: %v", err)
		} else if info.Mode().Perm()&0200 == 0 {
			return fmt.Errorf("日志目录 %s 不可写", logDir)
		}
	}

	if AppConfig.Queue.Type != "memory" && AppConfig.Queue.Type != "leveldb" {
		return fmt.Errorf("不支持的队列类型: %s，支持的类型: memory, leveldb", AppConfig.Queue.Type)
	}

	if AppConfig.Queue.Type == "leveldb" && AppConfig.Queue.Path == "" {
		return fmt.Errorf("使用 LevelDB 队列时必须配置存储路径")
	}

	return nil
}

// maskToken 对敏感信息进行掩码处理
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// maskURL 对 URL 进行掩码处理
func maskURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return "invalid-url"
	}
	return fmt.Sprintf("%s://%s/***", u.Scheme, u.Host)
}
