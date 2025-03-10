package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigPath 默认配置文件路径
	DefaultConfigPath = "/etc/tg-forward/config.yaml"
)

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

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) error {
	// 如果未指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = GetConfigPath()
	}

	// 确保配置文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 确保配置文件目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		logrus.Errorf("无法读取配置文件: %v", err)
		return err
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		logrus.Errorf("无法解析配置: %v", err)
		return err
	}

	// 设置默认值
	if AppConfig.Metrics.Interval <= 0 {
		AppConfig.Metrics.Interval = 60 // 默认 60 秒
	}

	if AppConfig.Metrics.HTTP.Port <= 0 {
		AppConfig.Metrics.HTTP.Port = 9090 // 默认端口 9090
	}

	if AppConfig.Metrics.HTTP.Path == "" {
		AppConfig.Metrics.HTTP.Path = "/metrics" // 默认路径 /metrics
	}

	if AppConfig.Metrics.HTTP.HeaderName == "" {
		AppConfig.Metrics.HTTP.HeaderName = "X-API-Key" // 默认 API Key 请求头名称
	}

	// 设置 HTTPS 默认值
	if AppConfig.Metrics.HTTP.TLS.Enabled && AppConfig.Metrics.HTTP.TLS.Port <= 0 {
		AppConfig.Metrics.HTTP.TLS.Port = AppConfig.Metrics.HTTP.Port // 默认使用与 HTTP 相同的端口
	}

	logrus.Infof("已加载配置文件: %s", configPath)
	return nil
}
