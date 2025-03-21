package queue

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/user/tg-forward-to-xx/internal/config"
	"github.com/user/tg-forward-to-xx/internal/models"
	"github.com/sirupsen/logrus"
)

// LevelDBQueue 基于 LevelDB 的持久化队列实现
type LevelDBQueue struct {
	db        *leveldb.DB
	mutex     sync.Mutex
	indexKey  []byte
	closed    bool
	queuePath string
}

// 队列索引键
const indexKey = "queue:index"

var (
	// 确保只初始化一次
	initOnce sync.Once
)

// 初始化并注册 LevelDB 队列
func init() {
	initOnce.Do(func() {
		// 注册 LevelDB 队列工厂
		Register("leveldb", func() (Queue, error) {
			logrus.Debug("创建新的 LevelDB 队列实例")
			return createLevelDBQueue()
		})
	})
}

// NewLevelDBQueue 创建一个新的 LevelDB 队列
func NewLevelDBQueue() (Queue, error) {
	logrus.Debug("通过 NewLevelDBQueue 创建队列")
	return Create("leveldb")
}

// 创建 LevelDB 队列
func createLevelDBQueue() (Queue, error) {
	queuePath := config.AppConfig.Queue.Path
	logrus.Debugf("开始创建 LevelDB 队列，路径: %s", queuePath)

	// 检查目录是否存在
	dirInfo, err := os.Stat(queuePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，创建它
			logrus.Debugf("队列目录不存在，创建目录: %s", queuePath)
			if err := os.MkdirAll(queuePath, 0755); err != nil {
				return nil, fmt.Errorf("创建队列目录失败: %w", err)
			}
			// 重新获取目录信息
			dirInfo, err = os.Stat(queuePath)
			if err != nil {
				return nil, fmt.Errorf("检查新创建的队列目录状态失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("检查队列目录状态失败: %w", err)
		}
	}
	
	// 检查目录权限模式
	dirMode := dirInfo.Mode()
	logrus.Debugf("队列目录权限: %v", dirMode)
	if dirMode.Perm()&0700 != 0700 {
		return nil, fmt.Errorf("队列目录权限不足: %v", dirMode)
	}

	// 检查是否已有进程使用该数据库
	lockFile := filepath.Join(queuePath, "LOCK")
	if _, err := os.Stat(lockFile); err == nil {
		// 尝试打开锁文件，如果能打开则说明没有其他进程正在使用
		if file, err := os.OpenFile(lockFile, os.O_RDWR, 0666); err != nil {
			return nil, fmt.Errorf("数据库可能被其他进程锁定: %w", err)
		} else {
			file.Close()
			// 删除旧的锁文件
			if err := os.Remove(lockFile); err != nil {
				return nil, fmt.Errorf("删除旧的锁文件失败: %w", err)
			}
		}
	}

	// 打开 LevelDB 数据库，添加更多选项以提高稳定性
	options := &opt.Options{
		ErrorIfExist:   false,
		ErrorIfMissing: false,
		NoSync:         false,  // 启用同步写入
		NoWriteMerge:   false,  // 启用写入合并
		WriteBuffer:    64 * 1024 * 1024, // 64MB 写缓冲
		BlockCacheCapacity: 32 * 1024 * 1024, // 32MB 块缓存
	}
	
	logrus.Debug("尝试打开 LevelDB 数据库")
	db, err := leveldb.OpenFile(queuePath, options)
	if err != nil {
		// 如果打开失败，尝试清理可能的损坏文件
		if errors.IsCorrupted(err) {
			logrus.Warn("检测到数据库文件可能损坏，尝试恢复...")
			db, err = leveldb.RecoverFile(queuePath, nil)
			if err != nil {
				return nil, fmt.Errorf("恢复损坏的 LevelDB 失败: %w", err)
			}
			logrus.Info("数据库恢复成功")
		} else {
			return nil, fmt.Errorf("打开 LevelDB 失败: %w", err)
		}
	}

	queue := &LevelDBQueue{
		db:        db,
		indexKey:  []byte(indexKey),
		closed:    false,
		queuePath: queuePath,
	}

	// 初始化索引
	if _, err := db.Get(queue.indexKey, nil); err == leveldb.ErrNotFound {
		if err := db.Put(queue.indexKey, []byte("0"), nil); err != nil {
			db.Close()
			return nil, fmt.Errorf("初始化队列索引失败: %w", err)
		}
	}

	logrus.Info("LevelDB 队列创建成功")
	return queue, nil
}

// Push 将消息添加到队列
func (q *LevelDBQueue) Push(msg *models.Message) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	// 获取当前索引
	indexBytes, err := q.db.Get(q.indexKey, nil)
	if err != nil {
		return fmt.Errorf("获取队列索引失败: %w", err)
	}

	index, err := strconv.ParseInt(string(indexBytes), 10, 64)
	if err != nil {
		return fmt.Errorf("解析队列索引失败: %w", err)
	}

	// 序列化消息
	msgBytes, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 生成消息键
	msgKey := fmt.Sprintf("msg:%d", index)

	// 存储消息
	if err := q.db.Put([]byte(msgKey), msgBytes, nil); err != nil {
		return fmt.Errorf("存储消息失败: %w", err)
	}

	// 更新索引
	newIndex := strconv.FormatInt(index+1, 10)
	if err := q.db.Put(q.indexKey, []byte(newIndex), nil); err != nil {
		return fmt.Errorf("更新队列索引失败: %w", err)
	}

	return nil
}

// Pop 从队列中取出一条消息
func (q *LevelDBQueue) Pop() (*models.Message, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return nil, ErrQueueClosed
	}

	// 获取所有消息键
	keys, err := q.getMessageKeys()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ErrQueueEmpty
	}

	// 获取第一条消息
	msgBytes, err := q.db.Get([]byte(keys[0]), nil)
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %w", err)
	}

	// 解析消息
	msg, err := models.FromJSON(msgBytes)
	if err != nil {
		return nil, fmt.Errorf("解析消息失败: %w", err)
	}

	// 删除消息
	if err := q.db.Delete([]byte(keys[0]), nil); err != nil {
		return nil, fmt.Errorf("删除消息失败: %w", err)
	}

	return msg, nil
}

// Peek 查看队列中的下一条消息但不移除
func (q *LevelDBQueue) Peek() (*models.Message, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return nil, ErrQueueClosed
	}

	// 获取所有消息键
	keys, err := q.getMessageKeys()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ErrQueueEmpty
	}

	// 获取第一条消息
	msgBytes, err := q.db.Get([]byte(keys[0]), nil)
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %w", err)
	}

	// 解析消息
	msg, err := models.FromJSON(msgBytes)
	if err != nil {
		return nil, fmt.Errorf("解析消息失败: %w", err)
	}

	return msg, nil
}

// Size 返回队列中的消息数量
func (q *LevelDBQueue) Size() (int, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return 0, ErrQueueClosed
	}

	// 获取所有消息键
	keys, err := q.getMessageKeys()
	if err != nil {
		return 0, err
	}

	return len(keys), nil
}

// Close 关闭队列
func (q *LevelDBQueue) Close() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	return q.db.Close()
}

// 获取所有消息键
func (q *LevelDBQueue) getMessageKeys() ([]string, error) {
	var keys []string

	iter := q.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		if key != indexKey && len(key) > 4 && key[:4] == "msg:" {
			keys = append(keys, key)
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("遍历消息键失败: %w", err)
	}

	// 按照消息键排序
	sort.Strings(keys)

	return keys, nil
}
