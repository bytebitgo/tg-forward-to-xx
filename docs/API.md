# API 文档

## 基础信息

- 基础路径: `http://localhost:8080`
- 默认端口: 8080（可通过 `-http-port` 参数修改）
- 认证方式: 暂无认证要求
- 时间格式: ISO8601 格式（`YYYY-MM-DDThh:mm:ssZ`）
  - YYYY: 4位年份，如 2025
  - MM: 2位月份，01-12
  - DD: 2位日期，01-31
  - hh: 2位小时，00-23
  - mm: 2位分钟，00-59
  - ss: 2位秒数，00-59
  - Z: 表示 UTC 时间
  - 示例: `2025-03-11T00:12:00Z`

## API 端点

### 1. 查询聊天记录

#### 请求
- 方法: `GET`
- 路径: `/api/chat/history`
- 参数:
  - `chat_id`: 群组 ID（必填）
  - `start_time`: 开始时间（可选，格式：`2024-01-01T00:00:00Z`）
  - `end_time`: 结束时间（可选，格式：`2024-01-02T00:00:00Z`）
  - `limit`: 返回记录数量限制（可选，默认：100）
  - `offset`: 分页偏移量（可选，默认：0）

#### 请求示例

1. 基本查询：
```bash
curl "http://localhost:8080/api/chat/history?chat_id=123456789"
```

2. 带时间范围：
```bash
curl "http://localhost:8080/api/chat/history?chat_id=123456789&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z"
```

3. 分页查询：
```bash
curl "http://localhost:8080/api/chat/history?chat_id=123456789&limit=50&offset=100"
```

4. 完整参数：
```bash
curl "http://localhost:8080/api/chat/history?chat_id=123456789&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&limit=50&offset=0"
```

#### 响应示例
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "messages": [
      {
        "id": "123456789",
        "chat_id": 987654321,
        "username": "user123",
        "content": "消息内容",
        "created_at": "2024-01-01T12:00:00Z"
      }
    ]
  }
}
```

### 2. 按用户查询聊天记录

#### 请求
- 方法: `GET`
- 路径: `/api/chat/history/user`
- 参数:
  - `chat_id`: 群组 ID（必填）
  - `username`: 用户名（必填）
  - `start_time`: 开始时间（可选）
  - `end_time`: 结束时间（可选）
  - `limit`: 返回记录数量限制（可选，默认：100）
  - `offset`: 分页偏移量（可选，默认：0）

#### 请求示例

1. 基本用户查询：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123"
```

2. 带时间范围的用户查询：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z"
```

3. 用户消息分页：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123&limit=50&offset=100"
```