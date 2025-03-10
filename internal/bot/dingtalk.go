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
	// 构建消息内容
	content := fmt.Sprintf("[%s] %s: %s", msg.ChatTitle, msg.From, msg.Content)

	// 构建钉钉消息
	dingMsg := DingTalkMessage{
		MsgType: "text",
		Text: struct {
			Content string `json:"content"`
		}{
			Content: content,
		},
	}

	// 序列化消息
	msgBytes, err := json.Marshal(dingMsg)
	if err != nil {
		return fmt.Errorf("序列化钉钉消息失败: %w", err)
	}

	// 构建请求 URL（添加签名）
	reqURL, err := c.buildRequestURL()
	if err != nil {
		return fmt.Errorf("构建请求 URL 失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(msgBytes))
	if err != nil {
		return fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("钉钉 API 返回错误: %s", string(body))
	}

	logrus.Infof("成功发送消息到钉钉: %s", content)
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
