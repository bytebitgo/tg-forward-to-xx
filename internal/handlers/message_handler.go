package handlers

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/bot"
	"github.com/user/tg-forward-to-xx/internal/config"
	"github.com/user/tg-forward-to-xx/internal/metrics"
	"github.com/user/tg-forward-to-xx/internal/models"
	"github.com/user/tg-forward-to-xx/internal/queue"
	"github.com/user/tg-forward-to-xx/internal/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	dingTalk        *bot.DingTalkClient
	messageQueue    queue.Queue
	maxAttempts     int
	retryInterval   time.Duration
	stopChan        chan struct{}
	msgChan         chan *models.Message
	metricsReporter *metrics.Reporter
	bot             *tgbotapi.BotAPI
	storage         *storage.ChatHistoryStorage
	stopped         bool
}

// NewMessageHandler 创建一个新的消息处理器
func NewMessageHandler(q queue.Queue, storage *storage.ChatHistoryStorage) (*MessageHandler, error) {
	handler := &MessageHandler{
		dingTalk:      bot.NewDingTalkClient(),
		messageQueue:  q,
		maxAttempts:   config.AppConfig.Retry.MaxAttempts,
		retryInterval: time.Duration(config.AppConfig.Retry.Interval) * time.Second,
		stopChan:      make(chan struct{}),
		msgChan:       make(chan *models.Message, 100),
		storage:       storage,
		stopped:       false,
	}

	// 如果启用了指标收集，创建指标报告器
	if config.AppConfig.Metrics.Enabled {
		interval := time.Duration(config.AppConfig.Metrics.Interval) * time.Second
		handler.metricsReporter = metrics.NewReporter(q, interval, config.AppConfig.Metrics.OutputFile)
	}

	bot, err := tgbotapi.NewBotAPI(config.AppConfig.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("创建 Telegram 客户端失败: %w", err)
	}
	handler.bot = bot

	return handler, nil
}

// Start 启动消息处理器
func (h *MessageHandler) Start() error {
	logrus.Info("🔄 正在启动消息处理器...")

	// 启动消息处理协程
	go h.processQueueMessages()
	logrus.Info("✅ 消息处理协程已启动")

	// 启动重试协程
	go h.retryFailedMessages()
	logrus.Info("✅ 失败消息重试协程已启动")

	// 启动 Telegram 监听
	go func() {
		logrus.Info("🔄 正在启动 Telegram 消息监听...")
		updateConfig := tgbotapi.NewUpdate(0)
		updateConfig.Timeout = 60
		updates := h.bot.GetUpdatesChan(updateConfig)
		h.processTelegramUpdates(updates)
	}()

	// 如果启用了指标收集，启动指标报告器
	if h.metricsReporter != nil {
		h.metricsReporter.Start()
		logrus.WithFields(logrus.Fields{
			"interval": config.AppConfig.Metrics.Interval,
			"path":     config.AppConfig.Metrics.OutputFile,
		}).Info("📊 指标收集已启动")
	}

	logrus.Info("✅ 消息处理器启动成功")
	return nil
}

// Stop 停止消息处理器
func (h *MessageHandler) Stop() {
	if !h.stopped {
		h.stopped = true
		close(h.stopChan)
	}

	if err := h.messageQueue.Close(); err != nil {
		logrus.Errorf("关闭消息队列失败: %v", err)
	}

	// 停止指标报告器
	if h.metricsReporter != nil {
		h.metricsReporter.Stop()
		logrus.Info("指标收集已停止")
	}
}

// 处理消息队列中的消息
func (h *MessageHandler) processQueueMessages() {
	logrus.Info("消息处理协程开始运行")
	
	for {
		select {
		case <-h.stopChan:
			logrus.Info("消息处理协程收到停止信号")
			return
		case msg := <-h.msgChan:
			logrus.WithFields(logrus.Fields{
				"message_id": msg.ID,
				"from":      msg.From,
				"chat_id":   msg.ChatID,
			}).Info("收到新消息，准备发送到钉钉")

			startTime := time.Now()
			if err := h.sendToDingTalk(msg); err != nil {
				logrus.WithFields(logrus.Fields{
					"message_id": msg.ID,
					"error":     err,
				}).Error("发送消息到钉钉失败")

				// 更新尝试次数和最后尝试时间
				msg.Attempts++
				msg.LastAttempt = time.Now()

				// 添加到队列
				if err := h.messageQueue.Push(msg); err != nil {
					logrus.WithFields(logrus.Fields{
						"message_id": msg.ID,
						"error":     err,
					}).Error("添加消息到队列失败")
				} else {
					logrus.WithField("message_id", msg.ID).Info("消息已添加到重试队列")
					metrics.IncrementFailedMessages()
					metrics.IncrementRetryCount()
				}
			} else {
				logrus.WithFields(logrus.Fields{
					"message_id": msg.ID,
					"duration":  time.Since(startTime),
				}).Info("消息发送成功")
				metrics.IncrementProcessedMessages()
			}
			metrics.AddMessageLatency(time.Since(startTime))
		}
	}
}

// getGroupName 获取群组名称，如果为空则使用群组ID
func (h *MessageHandler) getGroupName(chat *tgbotapi.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	return fmt.Sprintf("群组(%d)", chat.ID)
}

// processTelegramUpdates 处理 Telegram 更新
func (h *MessageHandler) processTelegramUpdates(updates tgbotapi.UpdatesChannel) {
	logrus.Info("开始处理 Telegram 更新...")
	
	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				logrus.Debug("收到非消息更新，已忽略")
				continue
			}

			logrus.WithFields(logrus.Fields{
				"update_id":   update.UpdateID,
				"message_id":  update.Message.MessageID,
				"chat_id":    update.Message.Chat.ID,
				"chat_title": update.Message.Chat.Title,
				"from_user":  update.Message.From.UserName,
				"has_photo":  len(update.Message.Photo) > 0,
				"has_document": update.Message.Document != nil,
				"has_video":   update.Message.Video != nil,
				"has_audio":   update.Message.Audio != nil,
			}).Debug("收到新消息")

			// 检查是否是目标群组的消息
			if !h.isTargetChat(update.Message.Chat.ID) {
				logrus.WithField("chat_id", update.Message.Chat.ID).Debug("非目标群组消息，已忽略")
				continue
			}

			// 获取群组名称
			groupName := h.getGroupName(update.Message.Chat)

			// 保存聊天记录
			history := &models.ChatHistory{
				ID:        int64(update.Message.MessageID),
				ChatID:    update.Message.Chat.ID,
				Text:      update.Message.Text,
				FromUser:  update.Message.From.UserName,
				GroupName: groupName,
				Timestamp: time.Unix(int64(update.Message.Date), 0),
			}

			if err := h.storage.SaveMessage(history); err != nil {
				logrus.WithError(err).Error("保存聊天记录失败")
			}

			// 构建消息内容
			var content string
			var fileURL string

			// 处理不同类型的消息
			switch {
			case len(update.Message.Photo) > 0:
				logrus.Debug("处理图片消息")
				// 获取最大尺寸的图片
				photo := update.Message.Photo[len(update.Message.Photo)-1]
				file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
				if err != nil {
					logrus.WithError(err).Error("获取图片文件信息失败")
				} else {
					// 下载文件并上传到 S3
					fileURL, err = h.downloadAndUploadToS3(file, "photos", "image.jpg")
					if err != nil {
						logrus.WithError(err).Error("处理图片文件失败")
					} else {
						logrus.WithField("s3_url", fileURL).Debug("获取到 S3 图片 URL")
					}
				}
				content = "[图片]"
				if update.Message.Caption != "" {
					content = fmt.Sprintf("[图片] %s", update.Message.Caption)
				}

			case update.Message.Document != nil:
				logrus.Debug("处理文档消息")
				file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: update.Message.Document.FileID})
				if err != nil {
					logrus.WithError(err).Error("获取文档文件信息失败")
				} else {
					// 下载文件并上传到 S3
					fileURL, err = h.downloadAndUploadToS3(file, "documents", update.Message.Document.FileName)
					if err != nil {
						logrus.WithError(err).Error("处理文档文件失败")
					} else {
						logrus.WithField("s3_url", fileURL).Debug("获取到 S3 文档 URL")
					}
				}
				content = fmt.Sprintf("[文档: %s]", update.Message.Document.FileName)
				if update.Message.Caption != "" {
					content = fmt.Sprintf("[文档: %s] %s", update.Message.Document.FileName, update.Message.Caption)
				}

			case update.Message.Video != nil:
				logrus.Debug("处理视频消息")
				file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: update.Message.Video.FileID})
				if err != nil {
					logrus.WithError(err).Error("获取视频文件信息失败")
				} else {
					// 下载文件并上传到 S3
					fileURL, err = h.downloadAndUploadToS3(file, "videos", "video.mp4")
					if err != nil {
						logrus.WithError(err).Error("处理视频文件失败")
					} else {
						logrus.WithField("s3_url", fileURL).Debug("获取到 S3 视频 URL")
					}
				}
				content = "[视频]"
				if update.Message.Caption != "" {
					content = fmt.Sprintf("[视频] %s", update.Message.Caption)
				}

			case update.Message.Audio != nil:
				logrus.Debug("处理音频消息")
				file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: update.Message.Audio.FileID})
				if err != nil {
					logrus.WithError(err).Error("获取音频文件信息失败")
				} else {
					// 下载文件并上传到 S3
					fileURL, err = h.downloadAndUploadToS3(file, "audios", "audio.mp3")
					if err != nil {
						logrus.WithError(err).Error("处理音频文件失败")
					} else {
						logrus.WithField("s3_url", fileURL).Debug("获取到 S3 音频 URL")
					}
				}
				content = "[音频]"
				if update.Message.Caption != "" {
					content = fmt.Sprintf("[音频] %s", update.Message.Caption)
				}

			case update.Message.Text != "":
				content = update.Message.Text
			default:
				content = "[不支持的消息类型]"
			}

			// 构建发送者信息
			var sender string
			if update.Message.From.UserName != "" {
				sender = "@" + update.Message.From.UserName
			} else {
				sender = update.Message.From.FirstName
				if update.Message.From.LastName != "" {
					sender += " " + update.Message.From.LastName
				}
			}

			// 如果有文件 URL，使用 markdown 格式
			if fileURL != "" {
				content = fmt.Sprintf("### 【%s】[%s]\n%s\n![预览](%s)", 
					groupName, 
					sender, 
					content,
					fileURL,
				)
			} else {
				content = fmt.Sprintf("【%s】[%s]\n%s", groupName, sender, content)
			}

			// 创建消息对象
			msg := &models.Message{
				ID:        fmt.Sprintf("%d", update.Message.MessageID),
				Content:   content,
				From:     sender,
				ChatID:   update.Message.Chat.ID,
				ChatTitle: groupName,
				CreatedAt: time.Now(),
				IsMarkdown: fileURL != "", // 添加标记表示是否为 markdown 格式
			}

			// 发送到消息通道
			select {
			case h.msgChan <- msg:
				logrus.WithFields(logrus.Fields{
					"message_id": msg.ID,
					"chat_id":   msg.ChatID,
					"has_file":  fileURL != "",
				}).Debug("消息已加入处理队列")
			default:
				logrus.WithFields(logrus.Fields{
					"message_id": msg.ID,
					"chat_id":   msg.ChatID,
				}).Warn("消息通道已满，消息可能丢失")
			}

		case <-h.stopChan:
			logrus.Info("收到停止信号，停止处理 Telegram 更新")
			return
		}
	}
}

// isTargetChat 检查是否是目标群组
func (h *MessageHandler) isTargetChat(chatID int64) bool {
	for _, id := range config.AppConfig.Telegram.ChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

// forwardToDingTalk 转发消息到钉钉
func (h *MessageHandler) forwardToDingTalk(message *tgbotapi.Message) error {
	// 构建发送者信息
	var sender string
	if message.From.UserName != "" {
		sender = "@" + message.From.UserName
	} else {
		// 如果没有用户名，使用姓名
		sender = message.From.FirstName
		if message.From.LastName != "" {
			sender += " " + message.From.LastName
		}
	}

	// 获取群组名称
	groupName := h.getGroupName(message.Chat)

	// 构建消息内容
	var content string

	// 如果是回复消息，添加回复信息
	if message.ReplyToMessage != nil {
		var replyTo string
		if message.ReplyToMessage.From.UserName != "" {
			replyTo = "@" + message.ReplyToMessage.From.UserName
		} else {
			replyTo = message.ReplyToMessage.From.FirstName
			if message.ReplyToMessage.From.LastName != "" {
				replyTo += " " + message.ReplyToMessage.From.LastName
			}
		}

		// 添加回复的原始消息（最多显示100个字符，避免太长）
		replyText := message.ReplyToMessage.Text
		if len(replyText) > 100 {
			replyText = replyText[:97] + "..."
		}

		content = fmt.Sprintf("【%s】[%s 回复 %s]\n▶ %s\n-------------------\n%s",
			groupName,
			sender,
			replyTo,
			replyText,
			message.Text)
	} else {
		// 普通消息
		content = fmt.Sprintf("【%s】[%s]\n%s", groupName, sender, message.Text)
	}

	// 转换为钉钉消息格式
	msg := &models.Message{
		ID:      fmt.Sprintf("%d", message.MessageID),
		ChatID:  message.Chat.ID,
		From:    sender,
		Content: content,
	}

	// 发送到钉钉
	return h.dingTalk.SendMessage(msg)
}

// 重试失败的消息
func (h *MessageHandler) retryFailedMessages() {
	logrus.Info("失败消息重试协程开始运行")
	ticker := time.NewTicker(h.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			logrus.Info("失败消息重试协程收到停止信号")
			return
		case <-ticker.C:
			size, err := h.messageQueue.Size()
			if err != nil {
				logrus.WithError(err).Error("获取队列大小失败")
				continue
			}

			if size > 0 {
				logrus.WithField("queue_size", size).Info("开始处理重试队列中的消息")
				h.processQueuedMessages()
			} else {
				logrus.Debug("重试队列为空，无需处理")
			}
		}
	}
}

// 处理队列中的消息
func (h *MessageHandler) processQueuedMessages() {
	size, err := h.messageQueue.Size()
	if err != nil {
		logrus.Errorf("获取队列大小失败: %v", err)
		return
	}

	if size == 0 {
		return
	}

	logrus.Infof("开始处理队列中的 %d 条消息", size)

	for i := 0; i < size; i++ {
		// 从队列中取出消息
		msg, err := h.messageQueue.Pop()
		if err != nil {
			if err != queue.ErrQueueEmpty {
				logrus.Errorf("从队列中取出消息失败: %v", err)
			}
			break
		}

		// 检查重试次数
		if msg.Attempts >= h.maxAttempts {
			logrus.Warnf("消息 %s 已达到最大重试次数 (%d)，放弃重试", msg.ID, h.maxAttempts)
			continue
		}

		// 尝试发送消息
		startTime := time.Now()
		if err := h.sendToDingTalk(msg); err != nil {
			logrus.Errorf("重试发送消息到钉钉失败: %v", err)

			// 更新尝试次数和最后尝试时间
			msg.Attempts++
			msg.LastAttempt = time.Now()

			// 重新添加到队列
			if err := h.messageQueue.Push(msg); err != nil {
				logrus.Errorf("重新添加消息到队列失败: %v", err)
			}
			// 增加重试消息计数
			metrics.IncrementRetryMessages()
			// 增加重试计数
			metrics.IncrementRetryCount()
		} else {
			logrus.Infof("成功重试发送消息: %s (尝试次数: %d)", msg.ID, msg.Attempts)
			// 增加处理成功消息计数
			metrics.IncrementProcessedMessages()
		}
		// 记录消息处理延迟
		metrics.AddMessageLatency(time.Since(startTime))
	}
}

// 发送消息到钉钉
func (h *MessageHandler) sendToDingTalk(msg *models.Message) error {
	err := h.dingTalk.SendMessage(msg)

	// 检查是否是网络错误
	if err != nil {
		if _, ok := err.(net.Error); ok {
			return fmt.Errorf("网络错误: %w", err)
		}

		if opErr, ok := err.(*net.OpError); ok {
			return fmt.Errorf("网络操作错误: %w", opErr)
		}

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("网络超时: %w", err)
		}
	}

	return err
}

// 添加 downloadAndUploadToS3 函数
func (h *MessageHandler) downloadAndUploadToS3(file tgbotapi.File, category, filename string) (string, error) {
	logrus.WithFields(logrus.Fields{
		"file_id":   file.FileID,
		"category":  category,
		"filename":  filename,
	}).Debug("开始下载并上传文件到 S3")

	// 创建 S3 客户端
	s3Client, err := storage.NewS3Client()
	if err != nil {
		return "", fmt.Errorf("创建 S3 客户端失败: %w", err)
	}

	// 下载文件
	fileURL := file.Link(config.AppConfig.Telegram.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	// 生成唯一的对象名称
	timestamp := time.Now().Format("20060102150405")
	objectName := fmt.Sprintf("%s/%s_%s", category, timestamp, filename)

	// 上传到 S3
	s3URL, err := s3Client.UploadFile(resp.Body, objectName, resp.Header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("上传到 S3 失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"file_id":     file.FileID,
		"object_name": objectName,
		"s3_url":      s3URL,
	}).Debug("文件已成功上传到 S3")

	return s3URL, nil
}
