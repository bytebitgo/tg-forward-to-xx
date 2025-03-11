package queue

import (
	"sync"

	"github.com/user/tg-forward-to-xx/internal/models"
)

// MemoryQueue 基于内存的队列实现
type MemoryQueue struct {
	messages []*models.Message
	mutex    sync.Mutex
	closed   bool
}

// 注册内存队列工厂
func init() {
	Register("memory", func() (Queue, error) {
		return &MemoryQueue{
			messages: make([]*models.Message, 0),
			closed:   false,
		}, nil
	})
}

// NewMemoryQueue 创建一个新的内存队列
func NewMemoryQueue() (Queue, error) {
	return &MemoryQueue{
		messages: make([]*models.Message, 0),
		closed:   false,
	}, nil
}

// Push 将消息添加到队列
func (q *MemoryQueue) Push(msg *models.Message) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	q.messages = append(q.messages, msg)
	return nil
}

// Pop 从队列中取出一条消息
func (q *MemoryQueue) Pop() (*models.Message, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return nil, ErrQueueClosed
	}

	if len(q.messages) == 0 {
		return nil, ErrQueueEmpty
	}

	msg := q.messages[0]
	q.messages = q.messages[1:]
	return msg, nil
}

// Peek 查看队列中的下一条消息但不移除
func (q *MemoryQueue) Peek() (*models.Message, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return nil, ErrQueueClosed
	}

	if len(q.messages) == 0 {
		return nil, ErrQueueEmpty
	}

	return q.messages[0], nil
}

// Size 返回队列中的消息数量
func (q *MemoryQueue) Size() (int, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return 0, ErrQueueClosed
	}

	return len(q.messages), nil
}

// Close 关闭队列
func (q *MemoryQueue) Close() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.closed = true
	return nil
}
