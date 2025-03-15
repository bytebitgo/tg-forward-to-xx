package notifier

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/config"
)

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	config *config.FeishuConfig
}

// NewFeishuNotifier 创建飞书通知器
func NewFeishuNotifier(cfg *config.FeishuConfig) *FeishuNotifier {
	logrus.WithFields(logrus.Fields{
		"webhook_url": cfg.WebhookURL,
		"enable_at":   cfg.EnableAt,
		"is_at_all":   cfg.IsAtAll,
		"at_user_ids": cfg.AtUserIDs,
	}).Info("初始化飞书通知器")

	return &FeishuNotifier{
		config: cfg,
	}
}

// FeishuMessage 飞书消息结构
type FeishuMessage struct {
	Timestamp string      `json:"timestamp"` // 时间戳
	Sign      string      `json:"sign"`      // 签名
	MsgType   string      `json:"msg_type"`  // 消息类型
	Content   interface{} `json:"content"`   // 消息内容
}

// genSign 生成签名
func (n *FeishuNotifier) genSign(timestamp string) (string, error) {
	logrus.WithField("timestamp", timestamp).Debug("开始生成签名")

	// 签名拼接格式：timestamp + "\n" + secret
	stringToSign := timestamp + "\n" + n.config.Secret
	logrus.WithField("string_to_sign", stringToSign).Debug("待签名字符串")

	// SHA256 计算 HMAC
	h := hmac.New(sha256.New, []byte(n.config.Secret))
	_, err := h.Write([]byte(stringToSign))
	if err != nil {
		return "", fmt.Errorf("写入签名数据失败: %v", err)
	}

	// Base64 编码
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	logrus.WithField("signature", signature).Debug("签名生成完成")
	return signature, nil
}

// Send 发送消息到飞书
func (n *FeishuNotifier) Send(title, content string, isFile bool, fileURL string) error {
	if !n.config.Enabled {
		logrus.Info("飞书通知已禁用")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"title":    title,
		"content":  content,
		"is_file":  isFile,
		"file_url": fileURL,
	}).Info("准备发送飞书消息")

	// 获取当前时间戳（秒级）
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// 生成签名
	sign, err := n.genSign(timestamp)
	if err != nil {
		logrus.WithError(err).Error("生成签名失败")
		return fmt.Errorf("生成签名失败: %v", err)
	}

	// 构建消息内容
	contentBlocks := make([][]map[string]interface{}, 1)
	contentBlocks[0] = make([]map[string]interface{}, 0)

	if isFile {
		// 文件消息
		contentBlocks[0] = append(contentBlocks[0], map[string]interface{}{
			"tag":  "text",
			"text": title + ": ",
		})
		// 如果是图片或文件，使用 tag=a 和实际的 S3 地址
		contentBlocks[0] = append(contentBlocks[0], map[string]interface{}{
			"tag":  "a",
			"text": "查看图片",
			"href": fileURL,
		})
	} else {
		// 纯文本消息
		contentBlocks[0] = append(contentBlocks[0], map[string]interface{}{
			"tag":  "text",
			"text": title + ": " + content,
		})
	}

	// 添加 @ 功能
	if n.config.EnableAt {
		if n.config.IsAtAll {
			contentBlocks[0] = append(contentBlocks[0], map[string]interface{}{
				"tag":     "at",
				"user_id": "all",
			})
		} else {
			for _, userID := range n.config.AtUserIDs {
				contentBlocks[0] = append(contentBlocks[0], map[string]interface{}{
					"tag":     "at",
					"user_id": userID,
				})
			}
		}
	}

	msg := FeishuMessage{
		Timestamp: timestamp,
		Sign:      sign,
		MsgType:   "post",
		Content: map[string]interface{}{
			"post": map[string]interface{}{
				"zh_cn": map[string]interface{}{
					"title":   title, // 使用群组名或用户名作为标题
					"content": contentBlocks,
				},
			},
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("JSON编码失败")
		return fmt.Errorf("JSON编码失败: %v", err)
	}
	logrus.WithField("json_data", string(jsonData)).Debug("消息JSON数据")

	req, err := http.NewRequest("POST", n.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.WithError(err).Error("创建HTTP请求失败")
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	logrus.WithFields(logrus.Fields{
		"url":     n.config.WebhookURL,
		"headers": req.Header,
	}).Debug("发送HTTP请求")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("发送HTTP请求失败")
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("读取响应内容失败")
		return fmt.Errorf("读取响应内容失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"status_code":   resp.StatusCode,
		"response_body": string(body),
	}).Info("收到飞书响应")

	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("飞书请求失败")
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	if n.config.NotifyVerbose {
		logrus.WithFields(logrus.Fields{
			"title":    title,
			"content":  content,
			"is_file":  isFile,
			"file_url": fileURL,
			"response": string(body),
		}).Info("飞书消息发送成功")
	}

	return nil
} 