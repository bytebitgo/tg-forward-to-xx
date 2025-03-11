# API 文档

## 基础信息

- 基础路径: `http://localhost:8080`
- 默认端口: 8080（可通过 `-http-port` 参数修改）
- 认证方式: 暂无认证要求
- 时间格式: ISO8601 格式（`YYYY-MM-DDThh:mm:ssZ`）
  - YYYY: 4位年份，如 2024
  - MM: 2位月份，01-12
  - DD: 2位日期，01-31
  - hh: 2位小时，00-23
  - mm: 2位分钟，00-59
  - ss: 2位秒数，00-59
  - Z: 表示 UTC 时间
  - 示例: `2024-03-11T00:00:00Z`

## 响应说明

### 成功响应
1. 有数据时：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "messages": [...]
  }
}
```

2. 无数据时：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 0,
    "messages": []
  }
}
```

### 错误响应
1. 参数错误（400 Bad Request）：
```
无效的开始时间 - start_time 参数缺失或格式错误
无效的结束时间 - end_time 参数格式错误
无效的群组ID - chat_id 参数缺失或格式错误
无效的用户名 - username 参数缺失或格式错误
```

### 常见问题处理
1. 返回 "无效的开始时间"：
   - start_time 是必填参数
   - 确保时间格式正确（YYYY-MM-DDThh:mm:ssZ）
   - 使用当前时间之前的时间范围
   - 建议使用最近24小时内的时间范围

2. 查询建议：
   - 必须提供 start_time 参数
   - 建议同时提供 end_time 参数
   - 时间范围不要超过7天
   - 检查用户名大小写是否正确

## API 端点

### 1. 查询聊天记录

#### 请求
- 方法: `GET`
- 路径: `/api/chat/history`
- 参数:
  - `chat_id`: 群组 ID（必填）
  - `start_time`: 开始时间（必填，格式：`2024-03-11T00:00:00Z`）
  - `end_time`: 结束时间（可选，默认为当前时间）
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
  - `start_time`: 开始时间（必填，格式：`2024-03-11T00:00:00Z`）
  - `end_time`: 结束时间（可选，默认为当前时间）
  - `limit`: 返回记录数量限制（可选，默认：100）
  - `offset`: 分页偏移量（可选，默认：0）

#### 请求示例

1. 基本用户查询（必须包含开始时间）：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123&start_time=2024-03-11T00:00:00Z"
```

2. 带时间范围的用户查询：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123&start_time=2024-03-11T00:00:00Z&end_time=2024-03-11T23:59:59Z"
```

3. 用户消息分页：
```bash
curl "http://localhost:8080/api/chat/history/user?chat_id=123456789&username=user123&limit=50&offset=100"
```

### 3. 导出聊天记录

#### 请求
- 方法: `GET`
- 路径: `/api/chat/history/export`
- 参数:
  - `chat_id`: 群组 ID（必填）
  - `username`: 用户名（可选，指定则只导出该用户的消息）
  - `start_time`: 开始时间（必填，格式：`2024-01-01T00:00:00Z`）
  - `end_time`: 结束时间（必填，格式：`2024-03-11T23:59:59Z`）
  - `format`: 导出格式（可选，默认：csv）

#### 请求示例

1. 导出指定时间范围的所有记录：
```bash
curl -o chat_history.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789&start_time=2024-01-01T00:00:00Z&end_time=2024-03-11T23:59:59Z"
```

2. 导出指定用户在指定时间范围的记录：
```bash
curl -o user_messages.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789&username=user123&start_time=2024-01-01T00:00:00Z&end_time=2024-03-11T23:59:59Z"
```

3. 导出最近一周的记录：
```bash
curl -o recent_messages.csv "http://localhost:8080/api/chat/history/export?chat_id=123456789&start_time=2024-03-04T00:00:00Z&end_time=2024-03-11T23:59:59Z"
```

#### 响应格式
- Content-Type: `text/csv`
- 文件内容：UTF-8 编码的 CSV 文件，包含 BOM 头
- 列信息：
  ```csv
  Message ID,Chat ID,Username,Content,Created At,Group Name
  123456789,-4646308813,user123,"消息内容","2024-03-11T12:00:00Z","群组名称"
  ```

#### 导出说明
1. 导出功能特点：
   - 必须指定时间范围（start_time 和 end_time）
   - 支持按用户筛选
   - CSV 格式方便在 Excel 中查看和处理
   - 包含完整的消息内容和时间戳

2. 使用建议：
   - 建议每次导出不超过3个月的数据
   - 如果数据量较大，可以按周或按月分批导出
   - 导出时间范围建议不要跨度太大，避免请求超时
   - 可以用 Excel 的筛选功能进行进一步分析

3. 时间范围说明：
   - start_time：必须指定，且不能是未来时间
   - end_time：必须指定，且要大于 start_time
   - 时间格式必须符合 ISO8601 标准
   - 建议按实际数据量调整时间范围