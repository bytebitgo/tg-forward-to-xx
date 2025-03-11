package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/models"
)

// TelegramClient Telegram 机器人客户端
type TelegramClient struct {
	bot     *tgbotapi.BotAPI
	chatIDs map[int64]bool
}

// NewTelegramClient 创建一个新的 Telegram 机器人客户端
func NewTelegramClient() (*TelegramClient, error) {
	bot, err := tgbotapi.NewBotAPI(config.AppConfig.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("创建 Telegram 机器人失败: %w", err)
	}

	// 创建聊天 ID 映射
	chatIDs := make(map[int64]bool)
	for _, id := range config.AppConfig.Telegram.ChatIDs {
		chatIDs[id] = true
	}

	return &TelegramClient{
		bot:     bot,
		chatIDs: chatIDs,
	}, nil
}

// StartListening 开始监听消息
func (c *TelegramClient) StartListening(msgChan chan<- *models.Message) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.bot.GetUpdatesChan(u)

	logrus.Info("开始监听 Telegram 消息")
	logrus.Infof("已配置的监听聊天 ID 列表: %v", c.chatIDs)

	for update := range updates {
		// 处理消息
		if update.Message != nil {
			// 打印详细的消息信息
			logrus.WithFields(logrus.Fields{
				"chat_id":      update.Message.Chat.ID,
				"chat_type":    update.Message.Chat.Type,
				"chat_title":   update.Message.Chat.Title,
				"from_id":      update.Message.From.ID,
				"from_name":    fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
				"from_user":    update.Message.From.UserName,
				"message_id":   update.Message.MessageID,
				"message_text": update.Message.Text,
			}).Info("收到新消息")

			// 检查是否是监听的聊天 ID
			if _, ok := c.chatIDs[update.Message.Chat.ID]; !ok {
				logrus.WithFields(logrus.Fields{
					"chat_id":    update.Message.Chat.ID,
					"chat_title": update.Message.Chat.Title,
					"chat_type":  update.Message.Chat.Type,
				}).Warn("⚠️ 此消息来自未配置的聊天，将被忽略。如需转发此聊天的消息，请在配置文件中添加此 chat_id")
				continue
			}

			c.handleMessage(update.Message, msgChan)
		}
	}

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
	if content == "" {
		if message.Caption != "" {
			content = message.Caption
		} else if message.Sticker != nil {
			content = "[贴纸]"
		} else if message.Photo != nil {
			content = "[图片]"
		} else if message.Document != nil {
			content = fmt.Sprintf("[文件: %s]", message.Document.FileName)
		} else if message.Audio != nil {
			content = "[音频]"
		} else if message.Video != nil {
			content = "[视频]"
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

// 截断字符串
func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
