package models

import (
	"encoding/json"
	"time"
)

// Message 表示从 Telegram 转发到钉钉的消息
type Message struct {
	ID          int64     `json:"id"`           // 唯一标识符
	Content     string    `json:"content"`      // 消息内容
	From        string    `json:"from"`         // 发送者
	ChatID      int64     `json:"chat_id"`      // 聊天ID
	ChatTitle   string    `json:"chat_title"`   // 聊天标题
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	Attempts    int       `json:"attempts"`     // 尝试次数
	LastAttempt time.Time `json:"last_attempt"` // 最后一次尝试时间
	IsMarkdown  bool      `json:"is_markdown"`  // 是否为 markdown 格式
}

// NewMessage 创建一个新的消息
func NewMessage(content, from string, chatID int64, chatTitle string) *Message {
	return &Message{
		ID:        time.Now().UnixNano(),  // 使用纳秒时间戳作为唯一标识符
		Content:   content,
		From:      from,
		ChatID:    chatID,
		ChatTitle: chatTitle,
		CreatedAt: time.Now(),
		Attempts:  0,
	}
}

// ToJSON 将消息转换为 JSON 字符串
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从 JSON 字符串解析消息
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// 生成唯一ID
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(b)
}
