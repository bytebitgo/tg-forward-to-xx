# API 文档

## 基础信息

- 基础路径: `http://localhost:8080`
- 默认端口: 8080（可通过 `-http-port` 参数修改）
- 认证方式: 暂无认证要求

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

#### 响应
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

#### 响应
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
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

### 3. 导出聊天记录

#### 请求
- 方法: `GET`
- 路径: `/api/chat/history/export`
- 参数:
  - `chat_id`: 群组 ID（必填）
  - `username`: 用户名（可选，指定则只导出该用户的消息）
  - `start_time`: 开始时间（可选）
  - `end_time`: 结束时间（可选）
  - `format`: 导出格式（可选，默认：csv）

#### 响应
- Content-Type: `text/csv`
- 文件内容：UTF-8 编码的 CSV 文件，包含 BOM 头
- 列信息：
  - Message ID
  - Chat ID
  - Username
  - Content
  - Created At

## 错误码说明

- 0: 成功
- 400: 请求参数错误
- 404: 资源不存在
- 500: 服务器内部错误

## 指标监控 API

### 1. 获取系统指标

#### 请求
- 方法: `GET`
- 路径: `/metrics`（默认端口：9090，可通过 `-metrics-port` 参数修改）
- 参数: 无

#### 响应
```json
{
  "queue": {
    "size": 10,
    "processed": 1000,
    "failed": 5,
    "retry": 2
  },
  "performance": {
    "avg_process_time": 0.5,
    "p95_process_time": 1.2,
    "messages_per_minute": 60
  },
  "uptime": "24h10m5s"
}
``` 