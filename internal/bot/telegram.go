package bot

import (
	"fmt"
	"path/filepath"
	"strings"

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
	// 获取发送者信息
	from := message.From.UserName
	if from == "" {
		from = fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName)
	}

	// 获取消息内容
	content := message.Text
	var fileURL string
	var err error

	if content == "" {
		if message.Caption != "" {
			content = message.Caption
		}

		// 处理文件、图片和视频
		if message.Document != nil {
			fileURL, err = c.handleFile(message.Document.FileID, "documents", message.Document.FileName, message.Document.MimeType)
			if err != nil {
				logrus.Errorf("处理文件失败: %v", err)
				content = fmt.Sprintf("[文件: %s (处理失败)]", message.Document.FileName)
			} else {
				content = fmt.Sprintf("[文件: %s]\n%s", message.Document.FileName, fileURL)
			}
		} else if message.Photo != nil && len(message.Photo) > 0 {
			// 获取最大尺寸的图片
			photo := message.Photo[len(message.Photo)-1]
			fileURL, err = c.handleFile(photo.FileID, "images", fmt.Sprintf("%d.jpg", message.MessageID), "image/jpeg")
			if err != nil {
				logrus.Errorf("处理图片失败: %v", err)
				content = "[图片 (处理失败)]"
			} else {
				content = fmt.Sprintf("[图片]\n%s", fileURL)
			}
		} else if message.Video != nil {
			fileURL, err = c.handleFile(message.Video.FileID, "videos", fmt.Sprintf("%d.mp4", message.MessageID), "video/mp4")
			if err != nil {
				logrus.Errorf("处理视频失败: %v", err)
				content = "[视频 (处理失败)]"
			} else {
				content = fmt.Sprintf("[视频]\n%s", fileURL)
			}
		} else if message.Sticker != nil {
			content = "[贴纸]"
		} else if message.Audio != nil {
			fileURL, err = c.handleFile(message.Audio.FileID, "audios", message.Audio.FileName, message.Audio.MimeType)
			if err != nil {
				logrus.Errorf("处理音频失败: %v", err)
				content = "[音频 (处理失败)]"
			} else {
				content = fmt.Sprintf("[音频: %s]\n%s", message.Audio.FileName, fileURL)
			}
		} else if message.Voice != nil {
			content = "[语音]"
		} else if message.VideoNote != nil {
			content = "[视频留言]"
		} else if message.Contact != nil {
			content = "[联系人]"
		} else if message.Location != nil {
			content = "[位置]"
		} else if message.Venue != nil {
			content = "[地点]"
		} else if message.Poll != nil {
			content = "[投票]"
		} else if message.Dice != nil {
			content = "[骰子]"
		} else {
			content = "[未知消息类型]"
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
	}).Info("✅ 消息已确认，准备转发")

	// 发送到消息通道
	select {
	case msgChan <- msg:
		logrus.WithField("message_id", msg.ID).Debug("消息已加入处理队列")
	default:
		logrus.WithField("message_id", msg.ID).Warn("消息通道已满，消息可能丢失")
	}
}

// handleFile 处理文件上传
func (c *TelegramClient) handleFile(fileID string, category string, filename string, contentType string) (string, error) {
	// 获取文件信息
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 下载文件
	fileURL := file.Link(c.bot.Token)
	resp, err := utils.HTTPClient.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	// 生成 S3 对象名称
	objectName := filepath.Join(category, filename)

	// 上传到 S3
	return c.s3Client.UploadFile(resp.Body, objectName, contentType)
}

// 截断字符串
func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
