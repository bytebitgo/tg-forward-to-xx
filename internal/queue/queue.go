package queue

import (
	"github.com/user/tg-forward-to-xx/internal/models"
)

// Queue 定义消息队列接口
type Queue interface {
	// Push 将消息添加到队列
	Push(msg *models.Message) error

	// Pop 从队列中取出一条消息
	Pop() (*models.Message, error)

	// Peek 查看队列中的下一条消息但不移除
	Peek() (*models.Message, error)

	// Size 返回队列中的消息数量
	Size() (int, error)

	// Close 关闭队列
	Close() error
}

// Factory 创建队列的工厂函数类型
type Factory func() (Queue, error)

// 注册的队列工厂
var queueFactories = make(map[string]Factory)

// Register 注册队列工厂
func Register(name string, factory Factory) {
	queueFactories[name] = factory
}

// Create 创建指定类型的队列
func Create(queueType string) (Queue, error) {
	if factory, ok := queueFactories[queueType]; ok {
		return factory()
	}
	return nil, ErrUnsupportedQueueType
}
