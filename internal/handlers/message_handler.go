package handlers

import (
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
	"github.com/user/tg-forward-to-xx/internal/bot"
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

// 处理 Telegram 更新
func (h *MessageHandler) processTelegramUpdates(updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			// 检查是否是目标群组的消息
			if !h.isTargetChat(update.Message.Chat.ID) {
				continue
			}

			// 保存聊天记录
			history := &models.ChatHistory{
				ID:        int64(update.Message.MessageID), // 转换为 int64
				ChatID:    update.Message.Chat.ID,
				Text:      update.Message.Text,
				FromUser:  update.Message.From.UserName,
				Timestamp: time.Unix(int64(update.Message.Date), 0),
			}

			if err := h.storage.SaveMessage(history); err != nil {
				logrus.Errorf("保存聊天记录失败: %v", err)
			}

			// 转发消息到钉钉
			if err := h.forwardToDingTalk(update.Message); err != nil {
				logrus.Errorf("转发消息失败: %v", err)
			}

		case <-h.stopChan:
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
	// 实现钉钉转发逻辑
	return nil
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
