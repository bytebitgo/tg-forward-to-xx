# Telegram 转发到钉钉

一个用 Golang 实现的应用程序，用于将 Telegram 群聊消息转发到钉钉机器人。

**当前版本：v1.0.4**

## 功能特点

- 监听指定的 Telegram 群聊消息
- 将消息转发到钉钉机器人
- 处理网络超时和错误情况
- 支持消息重试机制
- 支持持久化存储失败消息，程序重启后不会丢失
- 支持 systemd 和 SysV init 服务管理
- 支持 RPM 和 DEB 包安装
- 支持队列指标收集，便于对接 Prometheus 监控
- 提供 HTTP 接口暴露队列指标数据

## 安装

### 使用 Makefile 安装

项目提供了 Makefile 简化构建和安装过程：

```bash
# 构建应用
make build

# 安装到系统
sudo make install

# 卸载
sudo make uninstall

# 查看所有可用命令
make help
```

### 使用 RPM 包安装 (CentOS/RHEL/Fedora)

```bash
# 安装 RPM 包
sudo rpm -ivh tg-forward-1.0.4-1.el8.x86_64.rpm

# 编辑配置文件
sudo vi /etc/tg-forward/config.yaml

# 启动服务
sudo systemctl start tg-forward
```

### 使用 DEB 包安装 (Debian/Ubuntu)

```bash
# 安装 DEB 包
sudo dpkg -i tg-forward_1.0.4-1_amd64.deb

# 编辑配置文件
sudo vi /etc/tg-forward/config.yaml

# 启动服务
sudo systemctl start tg-forward
```

### 从源码构建

```bash
git clone https://github.com/user/tg-forward-to-xx.git
cd tg-forward-to-xx
go build -o tg-forward cmd/tgforward/main.go
```

### 使用 Docker

```bash
docker build -t tg-forward-to-xx .
docker run -v $(pwd)/config:/app/config -v $(pwd)/data:/app/data tg-forward-to-xx
```

## 配置

在运行程序前，需要先配置 `config/config.yaml` 文件：

```yaml
telegram:
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  chat_ids: [123456789, -987654321]  # 要监听的聊天ID列表

dingtalk:
  webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_ACCESS_TOKEN"
  secret: "YOUR_SECRET"  # 钉钉机器人安全设置中的签名密钥

queue:
  type: "leveldb"  # 可选: "memory" 或 "leveldb"
  path: "./data/queue"  # LevelDB 存储路径

retry:
  max_attempts: 5  # 最大重试次数
  interval: 60  # 重试间隔（秒）

metrics:
  enabled: true  # 是否启用指标收集
  interval: 60   # 收集间隔（秒）
  output_file: "./data/metrics/queue_metrics.json"  # 指标输出文件路径
  http:
    enabled: true  # 是否启用 HTTP 服务
    port: 9090     # HTTP 服务端口
    path: "/metrics"  # 指标 API 路径
```

### 获取 Telegram Bot Token

1. 在 Telegram 中搜索 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot` 命令创建一个新机器人
3. 按照提示完成创建，获取 API Token

### 获取 Telegram 聊天 ID

1. 将机器人添加到群组中
2. 发送一条消息到群组
3. 访问 `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
4. 在返回的 JSON 中找到 `chat` 对象中的 `id` 字段

### 创建钉钉机器人

1. 在钉钉群设置中添加自定义机器人
2. 选择安全设置（推荐使用"加签"）
3. 获取 Webhook URL 和签名密钥

## 使用方法

### 命令行运行

```bash
# 使用默认配置文件
./tg-forward

# 指定配置文件
./tg-forward -config /path/to/config.yaml

# 设置日志级别
./tg-forward -log-level debug
```

### 作为 systemd 服务运行

```bash
# 启动服务
sudo systemctl start tg-forward

# 停止服务
sudo systemctl stop tg-forward

# 查看服务状态
sudo systemctl status tg-forward

# 设置开机自启
sudo systemctl enable tg-forward
```

### 作为 SysV init 服务运行

```bash
# 启动服务
sudo service tg-forward start

# 停止服务
sudo service tg-forward stop

# 查看服务状态
sudo service tg-forward status

# 设置开机自启
sudo chkconfig tg-forward on
```

## 构建安装包

项目提供了构建 RPM 和 DEB 包的脚本：

```bash
# 构建所有包
cd deploy
bash build-packages.sh

# 只构建 RPM 包
bash build-packages.sh --rpm-only

# 只构建 DEB 包
bash build-packages.sh --deb-only

# 不更新版本号
bash build-packages.sh --no-version-update
```

## 日志级别

- `debug`: 详细调试信息
- `info`: 一般信息（默认）
- `warn`: 警告信息
- `error`: 错误信息

## 指标收集

应用程序支持收集队列相关指标，便于监控和对接 Prometheus 等监控系统。

### 指标配置

在 `config.yaml` 中配置指标收集：

```yaml
metrics:
  enabled: true  # 是否启用指标收集
  interval: 60   # 收集间隔（秒）
  output_file: "./data/metrics/queue_metrics.json"  # 指标输出文件路径
  http:
    enabled: true  # 是否启用 HTTP 服务
    port: 9090     # HTTP 服务端口
    path: "/metrics"  # 指标 API 路径
```

### 收集的指标

应用程序每分钟收集以下指标：

#### 基础指标
- `queue_size`: 当前队列中的消息数量
- `processed_messages`: 成功处理的消息数量
- `failed_messages`: 发送失败的消息数量
- `retry_messages`: 重试的消息数量
- `uptime_seconds`: 程序运行时间（秒）
- `last_update_time`: 最后更新时间

#### 性能指标
- `avg_latency_ms`: 消息平均处理延迟（毫秒）
- `p95_latency_ms`: 消息处理延迟的 P95 值（毫秒）
- `throughput_per_min`: 每分钟处理的消息数量
- `success_rate`: 消息处理成功率（百分比）
- `avg_retry_count`: 每条消息的平均重试次数
- `queue_pressure`: 队列积压程度（当前队列大小与处理速率的比值）
- `total_retry_count`: 总重试次数

### HTTP 指标接口

应用程序提供 HTTP/HTTPS 接口，可以通过以下方式访问指标数据：

```
http://your-server:9090/metrics
https://your-server:9443/metrics  # 如果启用了 HTTPS
```

#### HTTPS 配置

为了提供更安全的访问方式，可以启用 HTTPS：

```yaml
metrics:
  http:
    tls:
      enabled: true                    # 启用 HTTPS
      cert_file: "./certs/server.crt"  # 证书文件路径
      key_file: "./certs/server.key"   # 私钥文件路径
      port: 9443                       # HTTPS 端口（可选）
      force_https: true                # 强制使用 HTTPS
```

如果启用了 `force_https`，HTTP 请求将自动重定向到 HTTPS。

生成自签名证书（用于测试）：

```bash
# 生成私钥
openssl genrsa -out server.key 2048

# 生成证书签名请求
openssl req -new -key server.key -out server.csr

# 生成自签名证书
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
```

在生产环境中，建议使用受信任的 CA 签发的证书。

#### 认证配置

为了保护 HTTP/HTTPS 接口，可以启用 API Key 认证：

```yaml
metrics:
  http:
    auth: true                  # 启用认证
    api_key: "your-secret-key"  # 设置 API Key
    header_name: "X-API-Key"    # 自定义请求头名称（可选）
```

访问接口时需要在请求头中添加 API Key：

```bash
curl -H "X-API-Key: your-secret-key" http://your-server:9090/metrics
```

如果认证失败，服务器将返回 401 Unauthorized 状态码。

#### 返回数据示例

```json
{
  "queue_size": 10,
  "processed_messages": 1000,
  "failed_messages": 5,
  "retry_messages": 15,
  "last_update_time": "2024-01-09T10:00:00Z",
  "uptime_seconds": 3600,
  "avg_latency_ms": 150,
  "p95_latency_ms": 300,
  "throughput_per_min": 60,
  "success_rate": 99.5,
  "avg_retry_count": 0.015,
  "queue_pressure": 0.167,
  "total_retry_count": 15
}
```

此外，还提供健康检查接口（同样需要认证）：

```bash
curl -H "X-API-Key: your-secret-key" http://your-server:9090/health
```

### 对接 Prometheus

可以通过以下方式对接 Prometheus：

1. **使用 HTTP 接口**：在 Prometheus 配置中添加以下内容：

```yaml
scrape_configs:
  - job_name: 'tg-forward'
    scrape_interval: 60s
    metrics_path: '/metrics'
    static_configs:
      - targets: ['your-server:9090']
```

2. **使用文件**：如果不想暴露 HTTP 接口，可以使用 Prometheus 的 `file_sd_config` 或 `node_exporter` 的文本收集器功能读取指标文件。

### 监控建议

基于新增的指标，建议关注以下方面：

1. **性能监控**
   - 消息处理延迟（avg_latency_ms, p95_latency_ms）
   - 吞吐量（throughput_per_min）
   - 队列积压程度（queue_pressure）

2. **可靠性监控**
   - 消息处理成功率（success_rate）
   - 平均重试次数（avg_retry_count）
   - 失败消息数量（failed_messages）

3. **容量规划**
   - 队列大小趋势（queue_size）
   - 处理速率（throughput_per_min）
   - 积压程度（queue_pressure）

建议设置适当的告警阈值，例如：
- 当 P95 延迟超过 500ms
- 当成功率低于 99%
- 当队列积压程度超过 1.0
- 当每分钟重试次数异常增加

## 版本历史

查看 [CHANGELOG.md](CHANGELOG.md) 获取版本历史。

## 许可证

MIT