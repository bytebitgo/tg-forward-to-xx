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

	for update := range updates {
		// 处理消息
		if update.Message != nil {
			c.handleMessage(update.Message, msgChan)
		}
	}

	return nil
}

// 处理消息
func (c *TelegramClient) handleMessage(message *tgbotapi.Message, msgChan chan<- *models.Message) {
	// 检查消息是否来自配置的聊天
	if !c.chatIDs[message.Chat.ID] {
		return
	}

	// 获取消息内容
	content := message.Text

	// 如果消息为空，可能是媒体消息，尝试获取标题
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

	// 获取发送者信息
	var from string
	if message.From != nil {
		from = message.From.FirstName
		if message.From.LastName != "" {
			from += " " + message.From.LastName
		}
		if message.From.UserName != "" {
			from += " (@" + message.From.UserName + ")"
		}
	} else {
		from = "未知用户"
	}

	// 获取聊天标题
	chatTitle := message.Chat.Title
	if chatTitle == "" {
		if message.Chat.Type == "private" {
			chatTitle = "私聊"
		} else {
			chatTitle = "群聊"
		}
	}

	// 创建消息模型
	msg := models.NewMessage(content, from, message.Chat.ID, chatTitle)

	// 发送到消息通道
	msgChan <- msg

	logrus.Infof("收到来自 %s 的消息: %s", from, truncateString(content, 50))
}

// 截断字符串
func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
