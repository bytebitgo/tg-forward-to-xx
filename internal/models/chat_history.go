package models

import (
	"encoding/json"
	"time"
)

// ChatHistory 聊天记录结构
type ChatHistory struct {
	ID        int64     `json:"id"`         // 消息ID
	ChatID    int64     `json:"chat_id"`    // 群组ID
	Text      string    `json:"text"`       // 消息内容
	FromUser  string    `json:"from_user"`  // 发送者用户名
	Timestamp time.Time `json:"timestamp"`  // 消息时间戳
}

// ToJSON 将聊天记录转换为JSON
func (ch *ChatHistory) ToJSON() ([]byte, error) {
	return json.Marshal(ch)
}

// FromJSONHistory 从JSON转换为聊天记录
func FromJSONHistory(data []byte) (*ChatHistory, error) {
	var ch ChatHistory
	if err := json.Unmarshal(data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
} 