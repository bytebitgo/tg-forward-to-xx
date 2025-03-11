package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 首先检查当前目录
	if _, err := os.Stat("env.yaml"); err == nil {
		return "env.yaml"
	}

	// 然后检查 /etc/tg-forward/env.yaml
	if _, err := os.Stat("/etc/tg-forward/env.yaml"); err == nil {
		return "/etc/tg-forward/env.yaml"
	}

	// 最后检查用户主目录
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".tg-forward", "env.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// 如果都找不到，返回默认路径
	return "env.yaml"
}

// Config 应用配置结构
type Config struct {
	Telegram *TelegramConfig `mapstructure:"telegram"`
	Log      *LogConfig     `mapstructure:"log"`
	DingTalk *DingTalkConfig `mapstructure:"dingtalk"`
	Queue    *QueueConfig    `mapstructure:"queue"`
	Retry    *RetryConfig    `mapstructure:"retry"`
	Metrics  *MetricsConfig  `mapstructure:"metrics"`
	S3       *S3Config       `mapstructure:"s3"`
}

// TelegramConfig Telegram 配置
type TelegramConfig struct {
	Token   string  `mapstructure:"token"`    // Bot Token
	ChatIDs []int64 `mapstructure:"chat_ids"` // 要监听的聊天ID列表
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `mapstructure:"level"`     // 日志级别
	FilePath string `mapstructure:"file_path"` // 日志文件路径
	MaxSize  int    `mapstructure:"max_size"`  // 单个日志文件最大大小（MB）
	MaxFiles int    `mapstructure:"max_files"` // 最大保留文件数
}

// DingTalkConfig 钉钉机器人配置
type DingTalkConfig struct {
	WebhookURL    string   `mapstructure:"webhook_url"`    // Webhook URL
	Secret        string   `mapstructure:"secret"`         // 签名密钥
	EnableAt      bool     `mapstructure:"enable_at"`      // 是否启用 @ 功能
	AtMobiles     []string `mapstructure:"at_mobiles"`     // 需要 @ 的手机号列表
	IsAtAll       bool     `mapstructure:"is_at_all"`      // 是否 @ 所有人
	NotifyVerbose bool     `mapstructure:"notify_verbose"` // 是否显示详细信息
}

// QueueConfig 队列配置
type QueueConfig struct {
	Type string `mapstructure:"type"` // 队列类型：memory 或 leveldb
	Path string `mapstructure:"path"` // LevelDB 存储路径
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int `mapstructure:"max_attempts"` // 最大重试次数
	Interval    int `mapstructure:"interval"`     // 重试间隔（秒）
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // 是否启用指标收集
	Interval   int    `mapstructure:"interval"`    // 收集间隔（秒）
	OutputFile string `mapstructure:"output_file"` // 指标输出文件路径
	HTTP       *HTTPConfig `mapstructure:"http"`   // HTTP 服务配置
}

// HTTPConfig HTTP 服务配置
type HTTPConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // 是否启用 HTTP 服务
	Port       int    `mapstructure:"port"`        // HTTP 服务端口
	Path       string `mapstructure:"path"`        // 指标 API 路径
	Auth       bool   `mapstructure:"auth"`        // 是否启用认证
	APIKey     string `mapstructure:"api_key"`     // API Key
	HeaderName string `mapstructure:"header_name"` // API Key 请求头名称
	TLS        *TLSConfig `mapstructure:"tls"`     // TLS 配置
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // 是否启用 HTTPS
	CertFile   string `mapstructure:"cert_file"`   // 证书文件路径
	KeyFile    string `mapstructure:"key_file"`    // 私钥文件路径
	Port       int    `mapstructure:"port"`        // HTTPS 端口
	ForceHTTPS bool   `mapstructure:"force_https"` // 是否强制使用 HTTPS
}

// S3Config S3 配置
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`         // S3 端点
	Region          string `mapstructure:"region"`           // 区域
	Bucket          string `mapstructure:"bucket"`           // 存储桶名称
	AccessKeyID     string `mapstructure:"access_key_id"`    // 访问密钥 ID
	SecretAccessKey string `mapstructure:"secret_access_key"`// 访问密钥
	UseSSL          bool   `mapstructure:"use_ssl"`         // 是否使用 SSL
	PublicBaseURL   string `mapstructure:"public_base_url"` // 公共访问基础 URL
}

// AppConfig 全局配置实例
var AppConfig Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	logrus.WithField("path", configPath).Info("正在加载配置文件")

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 打印配置信息（隐藏敏感信息）
	logrus.WithFields(logrus.Fields{
		"telegram_token":    maskString(AppConfig.Telegram.Token),
		"telegram_chat_ids": AppConfig.Telegram.ChatIDs,
		"dingtalk_webhook": maskString(AppConfig.DingTalk.WebhookURL),
		"dingtalk_secret":  maskString(AppConfig.DingTalk.Secret),
		"queue_type":       AppConfig.Queue.Type,
		"queue_path":       AppConfig.Queue.Path,
	}).Info("配置加载完成")

	return nil
}

// maskString 隐藏敏感信息
func maskString(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
} 