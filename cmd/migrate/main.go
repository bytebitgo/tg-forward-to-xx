package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/models"
)

var (
	configFile string
	dryRun     bool
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/tg-forward/config.yaml", "配置文件路径")
	flag.BoolVar(&dryRun, "dry-run", false, "是否只预览变更而不实际执行")
	flag.Parse()
}

func sanitizeMessage(content string) string {
	// 检查是否包含无法解析的字符
	if strings.ContainsRune(content, '\uFFFD') {
		return "Emoji 解析失败"
	}
	return content
}

func main() {
	// 加载配置
	if err := config.LoadConfig(configFile); err != nil {
		logrus.Fatalf("加载配置文件失败: %v", err)
	}

	// 打开数据库
	dbPath := filepath.Join(config.AppConfig.Queue.Path, "chat_history")
	options := &opt.Options{
		ErrorIfExist:   false,
		ErrorIfMissing: false,
	}

	db, err := leveldb.OpenFile(dbPath, options)
	if err != nil {
		logrus.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 创建群组名称映射
	groupNames := make(map[int64]string)
	for _, chatID := range config.AppConfig.Telegram.ChatIDs {
		groupNames[chatID] = fmt.Sprintf("群组(%d)", chatID)
	}

	// 统计信息
	var total, updated int

	// 开始迭代所有记录
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)

	for iter.Next() {
		total++
		key := iter.Key()
		value := iter.Value()

		// 解析消息
		message, err := models.FromJSONHistory(value)
		if err != nil {
			logrus.Errorf("解析消息失败 [key=%x]: %v", key, err)
			continue
		}

		needsUpdate := false

		// 检查是否需要更新群组名称
		if message.GroupName == "" {
			message.GroupName = groupNames[message.ChatID]
			needsUpdate = true
		}

		// 检查消息内容是否需要处理
		sanitizedText := sanitizeMessage(message.Text)
		if sanitizedText != message.Text {
			message.Text = sanitizedText
			needsUpdate = true
		}

		if needsUpdate {
			updated++
			// 序列化消息
			newValue, err := message.ToJSON()
			if err != nil {
				logrus.Errorf("序列化消息失败 [id=%d]: %v", message.ID, err)
				continue
			}

			if !dryRun {
				// 添加到批处理
				batch.Put(key, newValue)

				// 每1000条记录执行一次批处理
				if updated%1000 == 0 {
					if err := db.Write(batch, nil); err != nil {
						logrus.Errorf("写入批处理失败: %v", err)
					}
					batch.Reset()
					logrus.Infof("已处理 %d/%d 条记录，更新 %d 条", total, total, updated)
				}
			}
		}
	}

	// 处理剩余的批处理
	if !dryRun && batch.Len() > 0 {
		if err := db.Write(batch, nil); err != nil {
			logrus.Errorf("写入最后的批处理失败: %v", err)
		}
	}

	if err := iter.Error(); err != nil {
		logrus.Errorf("迭代过程中发生错误: %v", err)
	}

	// 输出统计信息
	if dryRun {
		logrus.Infof("预览模式：共发现 %d 条记录，需要更新 %d 条", total, updated)
	} else {
		logrus.Infof("迁移完成：共处理 %d 条记录，更新 %d 条", total, updated)
	}
} 