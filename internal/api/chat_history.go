package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/storage"
)

// ChatHistoryHandler 聊天记录 API 处理器
type ChatHistoryHandler struct {
	storage *storage.ChatHistoryStorage
}

// NewChatHistoryHandler 创建新的聊天记录 API 处理器
func NewChatHistoryHandler(storage *storage.ChatHistoryStorage) *ChatHistoryHandler {
	return &ChatHistoryHandler{storage: storage}
}

// QueryHandler 处理聊天记录查询请求
func (h *ChatHistoryHandler) QueryHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数
	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil {
		http.Error(w, "无效的群组ID", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("start_time"))
	if err != nil {
		http.Error(w, "无效的开始时间", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("end_time"))
	if err != nil {
		http.Error(w, "无效的结束时间", http.StatusBadRequest)
		return
	}

	// 查询消息
	messages, err := h.storage.QueryMessages(chatID, startTime, endTime)
	if err != nil {
		http.Error(w, "查询失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 返回结果
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		http.Error(w, "编码响应失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// QueryByUserHandler 处理按用户查询聊天记录请求
func (h *ChatHistoryHandler) QueryByUserHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数
	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil {
		http.Error(w, "无效的群组ID", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "用户名不能为空", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("start_time"))
	if err != nil {
		http.Error(w, "无效的开始时间", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("end_time"))
	if err != nil {
		http.Error(w, "无效的结束时间", http.StatusBadRequest)
		return
	}

	// 查询消息
	messages, err := h.storage.QueryMessagesByUser(chatID, username, startTime, endTime)
	if err != nil {
		http.Error(w, "查询失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 返回结果
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		http.Error(w, "编码响应失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// ExportHandler 处理导出聊天记录请求
func (h *ChatHistoryHandler) ExportHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许 GET 请求
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数
	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil {
		http.Error(w, "无效的群组ID", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("start_time"))
	if err != nil {
		http.Error(w, "无效的开始时间", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("end_time"))
	if err != nil {
		http.Error(w, "无效的结束时间", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("username")

	// 创建临时文件
	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("chat_history_%d_%s.csv", chatID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(tmpDir, fileName)

	// 导出数据
	var exportErr error
	if username != "" {
		exportErr = h.storage.ExportUserToCSV(chatID, username, startTime, endTime, filePath)
	} else {
		exportErr = h.storage.ExportToCSV(chatID, startTime, endTime, filePath)
	}

	if exportErr != nil {
		http.Error(w, "导出失败: "+exportErr.Error(), http.StatusInternalServerError)
		return
	}

	// 读取文件内容
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "读取导出文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 删除临时文件
	defer os.Remove(filePath)

	// 设置响应头
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))

	// 发送文件内容
	if _, err := w.Write(fileContent); err != nil {
		logrus.Errorf("发送文件内容失败: %v", err)
	}
} 