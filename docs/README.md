# Telegram 消息转发到钉钉

## 功能特性

- 支持监听多个 Telegram 群组
- 支持转发消息到钉钉机器人
- 支持消息持久化存储
- 支持历史消息查询和导出
- 支持按用户筛选消息
- 支持系统监控和指标收集
- 支持日志文件管理
- 支持 HTTP API 接口

## 快速开始

### 1. 安装

```bash
# 下载最新版本
git clone https://github.com/your-username/tg-forward-to-xx.git
cd tg-forward-to-xx

# 编译
go build -o tg-forward cmd/tgforward/main.go
```

### 2. 配置

创建配置文件 `/etc/tg-forward/config.yaml`：

```yaml
telegram:
  token: "your_bot_token"
  chat_ids: [123456789]

dingtalk:
  webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=xxx"
  secret: "your_secret"

log:
  level: "info"
  file: "/var/log/tg-forward/main.log"

queue:
  type: "leveldb"
  path: "/var/lib/tg-forward/queue"
```

### 3. 运行

```bash
# 使用默认配置运行
./tg-forward

# 指定配置文件
./tg-forward -config /path/to/config.yaml

# 设置日志级别
./tg-forward -log-level debug

# 设置 HTTP 端口
./tg-forward -http-port 8080
```

### 4. 系统服务

创建 systemd 服务文件 `/etc/systemd/system/tg-forward.service`：

```ini
[Unit]
Description=Telegram to DingTalk Forward Service
After=network.target

[Service]
ExecStart=/usr/local/bin/tg-forward
Restart=always
User=tg-forward
Group=tg-forward

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable tg-forward
sudo systemctl start tg-forward
```

## 命令行参数

- `-config`: 配置文件路径，默认 `/etc/tg-forward/config.yaml`
- `-log-level`: 日志级别，可选 debug/info/warn/error
- `-http-port`: HTTP API 端口，默认 8080
- `-metrics-port`: 指标监控端口，默认 9090
- `-version`: 显示版本信息

## 日志管理

日志文件默认位置：`/var/log/tg-forward/main.log`

配置示例：
```yaml
log:
  level: "debug"  # 日志级别
  file: "/var/log/tg-forward/main.log"  # 日志文件路径
  max_size: 100   # 单个文件最大大小（MB）
  max_files: 5    # 最大保留文件数
```

## 监控指标

通过 HTTP API 获取系统指标：
```bash
curl http://localhost:9090/metrics
```

指标输出到文件：
```yaml
metrics:
  enabled: true
  interval: 60  # 收集间隔（秒）
  output_file: "/var/log/tg-forward/metrics.json"
```

## 常见问题

1. 队列访问错误
   - 检查目录权限
   - 确保没有其他实例占用
   - 删除损坏的 LOCK 文件

2. 日志级别设置
   - 命令行参数优先级高于配置文件
   - 支持动态调整日志级别

3. 性能优化
   - 使用 LevelDB 队列提高可靠性
   - 适当调整重试间隔
   - 监控系统资源使用

## 更新日志

详见 [CHANGELOG.md](./CHANGELOG.md)

## API 文档

详见 [API.md](./API.md)

## 架构设计

详见 [ARCHITECTURE.md](./ARCHITECTURE.md) 