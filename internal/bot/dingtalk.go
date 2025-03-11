package bot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/config"
	"github.com/user/tg-forward-to-xx/internal/models"
)

// DingTalkClient 钉钉机器人客户端
type DingTalkClient struct {
	webhookURL string
	secret     string
	atMobiles  []string
	isAtAll    bool
	httpClient *http.Client
}

// DingTalkMessage 钉钉消息结构
type DingTalkMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// NewDingTalkClient 创建一个新的钉钉机器人客户端
func NewDingTalkClient() *DingTalkClient {
	return &DingTalkClient{
		webhookURL: config.AppConfig.DingTalk.WebhookURL,
		secret:     config.AppConfig.DingTalk.Secret,
		atMobiles:  config.AppConfig.DingTalk.AtMobiles,
		isAtAll:    config.AppConfig.DingTalk.IsAtAll,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage 发送消息到钉钉
func (c *DingTalkClient) SendMessage(msg *models.Message) error {
	logrus.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"chat_id":   msg.ChatID,
		"from":      msg.From,
	}).Debug("准备发送消息到钉钉")

	// 构造消息体
	var data map[string]interface{}
	if msg.IsMarkdown {
		// Markdown 格式消息
		data = map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": fmt.Sprintf("来自 %s 的消息", msg.ChatTitle),
				"text":  msg.Content,
			},
		}
		// 只有在启用 @ 功能时才添加 at 字段
		if config.AppConfig.DingTalk.EnableAt {
			data["at"] = map[string]interface{}{
				"atMobiles": c.atMobiles,
				"isAtAll":   c.isAtAll,
			}
		}
	} else {
		// 普通文本消息
		content := msg.Content
		// 只有在启用 @ 功能时才添加 @ 信息
		if config.AppConfig.DingTalk.EnableAt && len(c.atMobiles) > 0 {
			content += "\n"
			for _, mobile := range c.atMobiles {
				content += fmt.Sprintf("@%s ", mobile)
			}
		}

		data = map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		}
		// 只有在启用 @ 功能时才添加 at 字段
		if config.AppConfig.DingTalk.EnableAt {
			data["at"] = map[string]interface{}{
				"atMobiles": c.atMobiles,
				"isAtAll":   c.isAtAll,
			}
		}
	}

	// 生成签名
	timestamp := time.Now().UnixMilli()
	sign := c.generateSign(timestamp)

	// 构造完整的 URL
	url := fmt.Sprintf("%s&timestamp=%d&sign=%s", c.webhookURL, timestamp, sign)

	logrus.WithField("url", url).Debug("发送 HTTP 请求到钉钉")

	// 发送请求
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"status":     resp.StatusCode,
		"response":   string(body),
	}).Debug("钉钉消息发送成功")

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("钉钉返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	return nil
}

// 构建带签名的请求 URL
func (c *DingTalkClient) buildRequestURL() (string, error) {
	if c.secret == "" {
		return c.webhookURL, nil
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	stringToSign := timestamp + "\n" + c.secret

	// 计算签名
	h := hmac.New(sha256.New, []byte(c.secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 解析原始 URL
	baseURL, err := url.Parse(c.webhookURL)
	if err != nil {
		return "", fmt.Errorf("解析 webhook URL 失败: %w", err)
	}

	// 添加签名参数
	query := baseURL.Query()
	query.Add("timestamp", timestamp)
	query.Add("sign", signature)
	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}

// generateSign 生成钉钉签名
func (c *DingTalkClient) generateSign(timestamp int64) string {
	signStr := fmt.Sprintf("%d\n%s", timestamp, c.secret)
	hmac256 := hmac.New(sha256.New, []byte(c.secret))
	hmac256.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))
	return url.QueryEscape(signature)
}
