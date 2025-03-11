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
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/models"
	"github.com/user/tg-forward-to-xx/internal/utils"
)

// DingTalkClient 钉钉机器人客户端
type DingTalkClient struct {
	webhookURL string
	secret     string
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

	// 处理消息内容中的表情
	sanitizedContent := utils.SanitizeMessage(msg.Content)

	// 构建消息内容
	content := fmt.Sprintf("来自 %s (%s):\n%s", msg.ChatTitle, msg.From, sanitizedContent)
	
	// 构建请求体
	reqBody := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"error":     err,
		}).Error("序列化消息失败")
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 获取当前时间戳
	timestamp := time.Now().UnixMilli()
	
	// 计算签名
	signStr := fmt.Sprintf("%d\n%s", timestamp, c.secret)
	hmac256 := hmac.New(sha256.New, []byte(c.secret))
	hmac256.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))

	// 构建完整的 URL
	url := fmt.Sprintf("%s&timestamp=%d&sign=%s", c.webhookURL, timestamp, url.QueryEscape(signature))

	logrus.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"url":       url,
	}).Debug("发送 HTTP 请求到钉钉")

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"error":     err,
		}).Error("创建 HTTP 请求失败")
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"error":     err,
		}).Error("发送 HTTP 请求失败")
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"error":     err,
		}).Error("读取响应失败")
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"error":     err,
			"response":  string(body),
		}).Error("解析响应失败")
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应状态
	if result.ErrCode != 0 {
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"error_code": result.ErrCode,
			"error_msg":  result.ErrMsg,
		}).Error("钉钉返回错误")
		return fmt.Errorf("钉钉返回错误: %s (错误码: %d)", result.ErrMsg, result.ErrCode)
	}

	logrus.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"status":    resp.StatusCode,
		"response":  string(body),
	}).Debug("钉钉消息发送成功")

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
