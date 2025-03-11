package utils

import "strings"

// SanitizeMessage 处理消息内容，清理无法解析的表情符号
func SanitizeMessage(content string) string {
	// 检查是否包含无法解析的字符
	if strings.ContainsRune(content, '\uFFFD') {
		return "Emoji 解析失败"
	}
	return content
} 