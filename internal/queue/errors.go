package queue

import "errors"

// 队列相关错误
var (
	ErrQueueEmpty           = errors.New("队列为空")
	ErrUnsupportedQueueType = errors.New("不支持的队列类型")
	ErrMessageNotFound      = errors.New("消息未找到")
	ErrQueueClosed          = errors.New("队列已关闭")
)
