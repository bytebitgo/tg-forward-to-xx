package storage

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/models"
)

// ChatHistoryStorage 聊天记录存储服务
type ChatHistoryStorage struct {
	db *leveldb.DB
}

// NewChatHistoryStorage 创建新的聊天记录存储服务
func NewChatHistoryStorage() (*ChatHistoryStorage, error) {
	dbPath := filepath.Join(config.AppConfig.Queue.Path, "chat_history")
	options := &opt.Options{
		ErrorIfExist:   false,
		ErrorIfMissing: false,
	}

	db, err := leveldb.OpenFile(dbPath, options)
	if err != nil {
		return nil, fmt.Errorf("打开聊天记录数据库失败: %w", err)
	}

	return &ChatHistoryStorage{db: db}, nil
}

// SaveMessage 保存聊天记录
func (s *ChatHistoryStorage) SaveMessage(history *models.ChatHistory) error {
	// 使用时间戳作为键的一部分
	key := makeKey(history.ChatID, history.Timestamp.UnixNano())
	
	// 序列化消息
	value, err := history.ToJSON()
	if err != nil {
		return fmt.Errorf("序列化聊天记录失败: %w", err)
	}

	// 存储消息
	if err := s.db.Put(key, value, nil); err != nil {
		return fmt.Errorf("存储聊天记录失败: %w", err)
	}

	return nil
}

// QueryMessages 查询指定时间范围内的聊天记录
func (s *ChatHistoryStorage) QueryMessages(chatID int64, start, end time.Time) ([]*models.ChatHistory, error) {
	var messages []*models.ChatHistory

	// 创建范围查询的起始和结束键
	startKey := makeKey(chatID, start.UnixNano())
	endKey := makeKey(chatID, end.UnixNano())

	// 创建范围迭代器
	iter := s.db.NewIterator(&util.Range{
		Start: startKey,
		Limit: endKey,
	}, nil)
	defer iter.Release()

	// 遍历结果
	for iter.Next() {
		value := iter.Value()
		message, err := models.FromJSONHistory(value)
		if err != nil {
			return nil, fmt.Errorf("解析聊天记录失败: %w", err)
		}
		messages = append(messages, message)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("遍历聊天记录失败: %w", err)
	}

	return messages, nil
}

// QueryMessagesByUser 查询指定用户在指定时间范围内的聊天记录
func (s *ChatHistoryStorage) QueryMessagesByUser(chatID int64, username string, start, end time.Time) ([]*models.ChatHistory, error) {
	messages, err := s.QueryMessages(chatID, start, end)
	if err != nil {
		return nil, err
	}

	var userMessages []*models.ChatHistory
	for _, msg := range messages {
		if msg.FromUser == username {
			userMessages = append(userMessages, msg)
		}
	}

	return userMessages, nil
}

// ExportToCSV 导出指定时间范围内的聊天记录到CSV文件
func (s *ChatHistoryStorage) ExportToCSV(chatID int64, start, end time.Time, filePath string) error {
	// 查询消息
	messages, err := s.QueryMessages(chatID, start, end)
	if err != nil {
		return fmt.Errorf("查询消息失败: %w", err)
	}

	// 创建CSV文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建CSV文件失败: %w", err)
	}
	defer file.Close()

	// 写入UTF-8 BOM
	file.Write([]byte{0xEF, 0xBB, 0xBF})

	// 创建CSV写入器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{"消息ID", "群组ID", "群组名称", "用户名", "消息内容", "时间"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("写入CSV表头失败: %w", err)
	}

	// 写入数据
	for _, msg := range messages {
		record := []string{
			fmt.Sprintf("%d", msg.ID),
			fmt.Sprintf("%d", msg.ChatID),
			msg.GroupName,
			msg.FromUser,
			msg.Text,
			msg.Timestamp.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("写入CSV数据失败: %w", err)
		}
	}

	return nil
}

// ExportUserToCSV 导出指定用户在指定时间范围内的聊天记录到CSV文件
func (s *ChatHistoryStorage) ExportUserToCSV(chatID int64, username string, start, end time.Time, filePath string) error {
	// 查询用户消息
	messages, err := s.QueryMessagesByUser(chatID, username, start, end)
	if err != nil {
		return fmt.Errorf("查询用户消息失败: %w", err)
	}

	// 创建CSV文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建CSV文件失败: %w", err)
	}
	defer file.Close()

	// 写入UTF-8 BOM
	file.Write([]byte{0xEF, 0xBB, 0xBF})

	// 创建CSV写入器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{"消息ID", "群组ID", "群组名称", "用户名", "消息内容", "时间"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("写入CSV表头失败: %w", err)
	}

	// 写入数据
	for _, msg := range messages {
		record := []string{
			fmt.Sprintf("%d", msg.ID),
			fmt.Sprintf("%d", msg.ChatID),
			msg.GroupName,
			msg.FromUser,
			msg.Text,
			msg.Timestamp.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("写入CSV数据失败: %w", err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (s *ChatHistoryStorage) Close() error {
	return s.db.Close()
}

// makeKey 生成存储键
// 格式: chat_id + timestamp，这样可以按时间顺序存储和查询
func makeKey(chatID int64, timestamp int64) []byte {
	key := make([]byte, 16)
	binary.BigEndian.PutUint64(key[:8], uint64(chatID))
	binary.BigEndian.PutUint64(key[8:], uint64(timestamp))
	return key
} 