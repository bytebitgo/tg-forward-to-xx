package bot

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/config"
)

// HarmonyClient HarmonyOS_MeoW 通知客户端
type HarmonyClient struct {
	enabled  bool
	userIDs  []string
	baseURL  string
	client   *http.Client
}

// NewHarmonyClient 创建新的 HarmonyOS_MeoW 客户端
func NewHarmonyClient() *HarmonyClient {
	cfg := config.AppConfig.Harmony
	if cfg == nil {
		return &HarmonyClient{enabled: false}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.chuckfang.com"
	}

	return &HarmonyClient{
		enabled:  cfg.Enabled,
		userIDs:  cfg.UserIDs,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// SendMessage 发送通知消息
func (c *HarmonyClient) SendMessage(chatName string, text string, imageURL string) error {
	if !c.enabled || len(c.userIDs) == 0 {
		return nil
	}

	var lastErr error
	for _, userID := range c.userIDs {
		// 构建通知 URL，不使用 url.PathEscape，直接拼接字符串
		notifyURL := fmt.Sprintf("%s/%s/%s/%s", 
			strings.TrimRight(c.baseURL, "/"),
			userID,
			chatName,
			text,
		)

		// 打印完整的请求 URL
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"title": chatName,
			"content": text,
			"full_url": notifyURL,
		}).Info("HarmonyOS_MeoW 请求 URL")

		// 发送 GET 请求
		resp, err := c.client.Get(notifyURL)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"user_id": userID,
				"title": chatName,
				"error": err,
				"url": notifyURL,
			}).Error("发送 HarmonyOS_MeoW 通知失败")
			lastErr = err
			continue
		}

		// 读取响应内容
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("通知发送失败，状态码：%d，响应：%s，URL：%s", 
				resp.StatusCode, 
				string(body),
				notifyURL,
			)
			logrus.WithFields(logrus.Fields{
				"user_id": userID,
				"title": chatName,
				"status_code": resp.StatusCode,
				"response": string(body),
				"url": notifyURL,
			}).Error("HarmonyOS_MeoW 通知返回错误状态码")
			lastErr = err
			continue
		}

		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"title": chatName,
			"status_code": resp.StatusCode,
			"response": string(body),
			"url": notifyURL,
		}).Debug("HarmonyOS_MeoW 通知发送成功")
	}

	return lastErr
} 