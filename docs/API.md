# API 文档

## 基础信息

- 基础路径: `http://localhost:8080`
- 默认端口: 8080（可通过 `-http-port` 参数修改）
- 认证方式: 暂无认证要求
- 时间格式: ISO8601 格式（`YYYY-MM-DDThh:mm:ssZ`）

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

4. 完整参数：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&limit=50&offset=0"
```

#### 响应示例
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

#### 请求示例

1. 导出全部聊天记录：
```bash
curl -o chat_history.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789"
```

2. 导出指定用户的聊天记录：
```bash
curl -o user_messages.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789&username=user123"
```

3. 导出指定时间范围的记录：
```bash
curl -o time_range.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z"
```

4. 完整参数导出：
```bash
curl -o full_export.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789&username=user123&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&format=csv"
```

#### 响应格式
- Content-Type: `text/csv`
- 文件内容：UTF-8 编码的 CSV 文件，包含 BOM 头
- 列信息：
  ```csv
  Message ID,Chat ID,Username,Content,Created At
  123456789,987654321,user123,"消息内容",2024-01-01T12:00:00Z
  ```

## 错误码说明

- 0: 成功
- 400: 请求参数错误
  ```json
  {
    "code": 400,
    "message": "invalid parameter: chat_id is required",
    "data": null
  }
  ```
- 404: 资源不存在
  ```json
  {
    "code": 404,
    "message": "chat history not found",
    "data": null
  }
  ```
- 500: 服务器内部错误
  ```json
  {
    "code": 500,
    "message": "internal server error",
    "data": null
  }
  ```

## 指标监控 API

### 1. 获取系统指标

#### 请求
- 方法: `GET`
- 路径: `/metrics`（默认端口：9090，可通过 `-metrics-port` 参数修改）
- 参数: 无

#### 请求示例
```bash
# 基本指标查询
curl "http://localhost:9090/metrics"

# 指定时间间隔的指标（如果支持）
curl "http://localhost:9090/metrics?interval=60"
```

#### 响应示例
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
  "uptime": "24h10m5s",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 使用建议

1. 时间范围查询
   - 建议使用合理的时间范围，避免查询过大的数据量
   - 时间格式必须符合 ISO8601 标准
   - 如果不指定时间范围，默认返回最近的记录

2. 分页查询
   - 建议使用合适的 `limit` 值，避免一次返回过多数据
   - 使用 `offset` 实现分页查询
   - 总记录数可以从响应的 `total` 字段获取

3. 导出功能
   - 大量数据导出建议使用时间范围限制
   - 导出文件默认使用 UTF-8 编码，支持中文
   - CSV 文件可以直接用 Excel 打开 