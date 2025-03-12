package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/config"
	"github.com/user/tg-forward-to-xx/internal/models"
)

// BarkClient Bark 通知客户端
type BarkClient struct {
	enabled    bool
	keys       []string
	sound      string
	icon       string
	httpClient *http.Client
}

// BarkMessage Bark 消息结构
type BarkMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Badge int    `json:"badge"`
	Sound string `json:"sound,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Group string `json:"group"`
}

// NewBarkClient 创建一个新的 Bark 通知客户端
func NewBarkClient() *BarkClient {
	// 如果 Bark 配置为空，创建一个禁用的客户端
	if config.AppConfig.Bark == nil {
		return &BarkClient{
			enabled:    false,
			keys:      []string{},
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	return &BarkClient{
		enabled:    config.AppConfig.Bark.Enabled,
		keys:       config.AppConfig.Bark.Keys,
		sound:      config.AppConfig.Bark.Sound,
		icon:       config.AppConfig.Bark.Icon,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage 发送消息到 Bark
func (c *BarkClient) SendMessage(chatName string, message *models.Message) error {
	if !c.enabled || len(c.keys) == 0 {
		return nil
	}

	barkMsg := &BarkMessage{
		Title: chatName,
		Body:  fmt.Sprintf("有来自%s的消息，请关注", chatName),
		Badge: 1,
		Sound: c.sound,
		Icon:  c.icon,
		Group: chatName,
	}

	jsonData, err := json.Marshal(barkMsg)
	if err != nil {
		return fmt.Errorf("序列化 Bark 消息失败: %w", err)
	}

	for _, key := range c.keys {
		url := fmt.Sprintf("https://api.day.app/%s", key)
		
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			logrus.Errorf("创建 Bark 请求失败 (key: %s): %v", key, err)
			continue
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			logrus.Errorf("发送 Bark 通知失败 (key: %s): %v", key, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logrus.Errorf("Bark 服务器返回错误 (key: %s): %d", key, resp.StatusCode)
			continue
		}

		logrus.WithFields(logrus.Fields{
			"chat_name": chatName,
			"key":       key[:4] + "****" + key[len(key)-4:],
		}).Debug("Bark 通知发送成功")
	}

	return nil
} 