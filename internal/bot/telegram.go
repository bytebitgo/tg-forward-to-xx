package bot

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/models"
	"github.com/user/tg-forward-to-xx/internal/storage"
	"github.com/user/tg-forward-to-xx/internal/utils"
)

// TelegramClient Telegram 机器人客户端
type TelegramClient struct {
	bot      *tgbotapi.BotAPI
	chatIDs  map[int64]bool
	s3Client *storage.S3Client
}

// NewTelegramClient 创建一个新的 Telegram 机器人客户端
func NewTelegramClient() (*TelegramClient, error) {
	token := config.AppConfig.Telegram.Token
	if token == "" {
		return nil, fmt.Errorf("Telegram Bot Token 未配置")
	}

	logrus.Info("🔄 正在连接到 Telegram API...")
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("创建 Telegram 机器人失败: %w", err)
	}

	// 创建 S3 客户端
	s3Client, err := storage.NewS3Client()
	if err != nil {
		return nil, fmt.Errorf("创建 S3 客户端失败: %w", err)
	}

	// 设置调试模式
	bot.Debug = true

	// 验证 Bot 信息
	logrus.WithFields(logrus.Fields{
		"bot_id":       bot.Self.ID,
		"bot_name":     bot.Self.UserName,
		"bot_is_bot":   bot.Self.IsBot,
		"auth_success": true,
	}).Info("✅ Telegram Bot 认证成功")

	// 创建聊天 ID 映射
	chatIDs := make(map[int64]bool)
	if len(config.AppConfig.Telegram.ChatIDs) == 0 {
		logrus.Warn("⚠️ 未配置任何聊天 ID，将不会转发任何消息")
	} else {
		for _, id := range config.AppConfig.Telegram.ChatIDs {
			chatIDs[id] = true
			logrus.WithField("chat_id", id).Info("➕ 添加监听聊天")
		}
	}

	return &TelegramClient{
		bot:      bot,
		chatIDs:  chatIDs,
		s3Client: s3Client,
	}, nil
}

// StartListening 开始监听消息
func (c *TelegramClient) StartListening(msgChan chan<- *models.Message) error {
	logrus.Info("🚀 初始化 Telegram 消息监听...")

	// 配置更新
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	
	logrus.WithFields(logrus.Fields{
		"timeout": u.Timeout,
		"offset":  u.Offset,
		"limit":   u.Limit,
	}).Debug("⚙️ Telegram 更新配置")

	// 获取更新通道
	updates := c.bot.GetUpdatesChan(u)
	logrus.Info("✅ 成功建立 Telegram 更新通道连接")

	// 打印监听配置
	logrus.WithFields(logrus.Fields{
		"chat_ids_count": len(c.chatIDs),
		"chat_ids":       c.chatIDs,
	}).Info("👂 开始监听 Telegram 消息")

	// 发送测试消息到日志
	logrus.Info("🤖 Bot 开始工作，等待消息...")

	for update := range updates {
		logrus.Debug("📥 收到更新事件")

		if update.Message == nil {
			logrus.Debug("⏭️ 收到非消息更新，已忽略")
			continue
		}

		// 打印详细的消息信息
		logrus.WithFields(logrus.Fields{
			"update_id":    update.UpdateID,
			"chat_id":      update.Message.Chat.ID,
			"chat_type":    update.Message.Chat.Type,
			"chat_title":   update.Message.Chat.Title,
			"from_id":      update.Message.From.ID,
			"from_name":    fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
			"from_user":    update.Message.From.UserName,
			"message_id":   update.Message.MessageID,
			"message_text": update.Message.Text,
			"message_date": update.Message.Time(),
		}).Info("📨 收到新消息")

		// 检查是否是监听的聊天 ID
		if _, ok := c.chatIDs[update.Message.Chat.ID]; !ok {
			logrus.WithFields(logrus.Fields{
				"chat_id":          update.Message.Chat.ID,
				"chat_title":       update.Message.Chat.Title,
				"chat_type":        update.Message.Chat.Type,
				"configured_chats": c.chatIDs,
			}).Warn("⚠️ 此消息来自未配置的聊天，将被忽略")
			continue
		}

		logrus.Debug("✅ 消息来自已配置的聊天，准备处理")
		c.handleMessage(update.Message, msgChan)
	}

	logrus.Warn("⚠️ Telegram 更新通道已关闭")
	return nil
}

// 处理消息
func (c *TelegramClient) handleMessage(message *tgbotapi.Message, msgChan chan<- *models.Message) {
	logrus.Debug("开始处理新消息...")

	// 获取发送者信息
	from := message.From.UserName
	if from == "" {
		from = fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName)
	}

	// 获取消息内容
	content := message.Text
	var fileURL string

	// 详细记录消息类型和内容
	logrus.WithFields(logrus.Fields{
		"message_id":    message.MessageID,
		"message_type":  getMessageType(message),
		"from":         from,
		"chat_id":      message.Chat.ID,
		"has_text":     message.Text != "",
		"has_caption":  message.Caption != "",
		"has_document": message.Document != nil,
		"has_photo":    message.Photo != nil,
		"has_video":    message.Video != nil,
		"has_audio":    message.Audio != nil,
	}).Debug("消息详情")

	// 处理文件类型的消息
	if message.Document != nil || message.Photo != nil || message.Video != nil || message.Audio != nil {
		logrus.Debug("检测到媒体文件，开始处理...")
		content, fileURL = c.processMediaMessage(message)
		if fileURL != "" {
			logrus.WithFields(logrus.Fields{
				"file_url": fileURL,
				"content":  content,
			}).Info("媒体文件处理完成")
		} else {
			logrus.Warn("媒体文件处理完成，但未获得文件URL")
		}
	} else {
		// 处理文本消息
		if content == "" {
			if message.Caption != "" {
				content = message.Caption
				logrus.WithField("caption", content).Debug("使用消息说明作为内容")
			} else {
				content = getDefaultContent(message)
				logrus.WithField("default_content", content).Debug("使用默认内容")
			}
		}
	}

	// 处理表情符号
	content = utils.SanitizeMessage(content)

	// 创建消息对象
	msg := models.NewMessage(
		content,
		from,
		message.Chat.ID,
		message.Chat.Title,
	)

	logrus.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"chat_id":   msg.ChatID,
		"from":      msg.From,
		"content":   msg.Content,
		"file_url":  fileURL,
	}).Info("✅ 消息已确认，准备转发")

	// 发送到消息通道
	select {
	case msgChan <- msg:
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"chat_id":   msg.ChatID,
		}).Debug("消息已加入处理队列")
	default:
		logrus.WithFields(logrus.Fields{
			"message_id": msg.ID,
			"chat_id":   msg.ChatID,
		}).Warn("消息通道已满，消息可能丢失")
	}
}

// processMediaMessage 处理媒体消息
func (c *TelegramClient) processMediaMessage(message *tgbotapi.Message) (content, fileURL string) {
	logrus.Debug("进入媒体消息处理函数")

	if message.Document != nil {
		logrus.WithFields(logrus.Fields{
			"file_id":   message.Document.FileID,
			"file_name": message.Document.FileName,
			"mime_type": message.Document.MimeType,
			"file_size": message.Document.FileSize,
		}).Debug("开始处理文档文件")
		
		var err error
		fileURL, err = c.handleFile(message.Document.FileID, "documents", message.Document.FileName, message.Document.MimeType)
		if err != nil {
			logrus.WithError(err).Error("处理文件失败")
			content = fmt.Sprintf("[文件: %s (处理失败)]", message.Document.FileName)
		} else {
			content = fmt.Sprintf("[文件: %s]\n%s", message.Document.FileName, fileURL)
			logrus.WithField("file_url", fileURL).Info("文档处理成功")
		}
		return
	}

	if message.Photo != nil && len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		logrus.WithFields(logrus.Fields{
			"file_id":     photo.FileID,
			"width":       photo.Width,
			"height":      photo.Height,
			"file_size":   photo.FileSize,
			"photo_count": len(message.Photo),
		}).Debug("开始处理图片文件")
		
		var err error
		fileURL, err = c.handleFile(photo.FileID, "images", fmt.Sprintf("%d.jpg", message.MessageID), "image/jpeg")
		if err != nil {
			logrus.WithError(err).Error("处理图片失败")
			content = "[图片 (处理失败)]"
		} else {
			content = fmt.Sprintf("[图片]\n%s", fileURL)
			logrus.WithField("file_url", fileURL).Info("图片处理成功")
		}
		return
	}

	if message.Video != nil {
		logrus.WithFields(logrus.Fields{
			"file_id":   message.Video.FileID,
			"duration":  message.Video.Duration,
			"width":     message.Video.Width,
			"height":    message.Video.Height,
			"mime_type": message.Video.MimeType,
			"file_size": message.Video.FileSize,
		}).Info("🎥 收到视频消息")
		
		var err error
		fileURL, err = c.handleFile(message.Video.FileID, "videos", fmt.Sprintf("%d.mp4", message.MessageID), "video/mp4")
		if err != nil {
			logrus.WithError(err).Error("处理视频失败")
			content = "[视频 (处理失败)]"
		} else {
			content = fmt.Sprintf("[视频]\n%s", fileURL)
		}
		return
	}

	if message.Audio != nil {
		logrus.WithFields(logrus.Fields{
			"file_id":   message.Audio.FileID,
			"duration":  message.Audio.Duration,
			"mime_type": message.Audio.MimeType,
			"file_size": message.Audio.FileSize,
		}).Info("🎵 收到音频消息")
		
		var err error
		fileURL, err = c.handleFile(message.Audio.FileID, "audios", message.Audio.FileName, message.Audio.MimeType)
		if err != nil {
			logrus.WithError(err).Error("处理音频失败")
			content = "[音频 (处理失败)]"
		} else {
			content = fmt.Sprintf("[音频: %s]\n%s", message.Audio.FileName, fileURL)
		}
		return
	}

	return
}

// getMessageType 获取消息类型
func getMessageType(message *tgbotapi.Message) string {
	switch {
	case message.Document != nil:
		return "document"
	case message.Photo != nil:
		return "photo"
	case message.Video != nil:
		return "video"
	case message.Audio != nil:
		return "audio"
	case message.Voice != nil:
		return "voice"
	case message.Sticker != nil:
		return "sticker"
	case message.Location != nil:
		return "location"
	case message.Text != "":
		return "text"
	default:
		return "unknown"
	}
}

// getDefaultContent 获取默认的消息内容
func getDefaultContent(message *tgbotapi.Message) string {
	switch {
	case message.Sticker != nil:
		return "[贴纸]"
	case message.Voice != nil:
		return "[语音]"
	case message.VideoNote != nil:
		return "[视频留言]"
	case message.Contact != nil:
		return "[联系人]"
	case message.Location != nil:
		return "[位置]"
	case message.Venue != nil:
		return "[地点]"
	case message.Poll != nil:
		return "[投票]"
	case message.Dice != nil:
		return "[骰子]"
	default:
		return "[未知消息类型]"
	}
}

// handleFile 处理文件上传
func (c *TelegramClient) handleFile(fileID string, category string, filename string, contentType string) (string, error) {
	logrus.WithFields(logrus.Fields{
		"file_id":      fileID,
		"category":     category,
		"filename":     filename,
		"content_type": contentType,
	}).Debug("开始文件处理流程")

	// 1. 获取文件信息
	logrus.Debug("正在获取文件信息...")
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := c.bot.GetFile(fileConfig)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file_id": fileID,
			"error":   err,
		}).Error("获取文件信息失败")
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	if file.FileSize == 0 {
		logrus.WithField("file_id", fileID).Warn("文件大小为0，可能是无效文件")
	}

	logrus.WithFields(logrus.Fields{
		"file_id":   fileID,
		"file_path": file.FilePath,
		"file_size": file.FileSize,
	}).Debug("成功获取文件信息")

	// 2. 获取下载链接并下载
	fileURL := file.Link(c.bot.Token)
	logrus.Debug("开始下载文件...")
	
	resp, err := utils.HTTPClient.Get(fileURL)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file_id": fileID,
			"error":   err,
		}).Error("下载文件失败")
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"status_code":    resp.StatusCode,
		"content_length": resp.ContentLength,
		"content_type":   resp.Header.Get("Content-Type"),
	}).Debug("文件下载状态")

	// 3. 生成 S3 对象名称
	timestamp := time.Now().Format("20060102150405")
	objectName := filepath.Join(category, fmt.Sprintf("%s_%s", timestamp, filename))
	
	logrus.WithField("object_name", objectName).Debug("准备上传到 S3")

	// 4. 上传到 S3
	logrus.Debug("开始上传到 S3...")
	s3URL, err := c.s3Client.UploadFile(resp.Body, objectName, contentType)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file_id":     fileID,
			"object_name": objectName,
			"error":       err,
		}).Error("上传到 S3 失败")
		return "", fmt.Errorf("上传到 S3 失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"object_name": objectName,
		"s3_url":     s3URL,
	}).Debug("文件处理完成")

	return s3URL, nil
}

// 截断字符串
func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
