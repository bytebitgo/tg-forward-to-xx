FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tg-forward ./cmd/tgforward

# 使用轻量级的 alpine 镜像
FROM alpine:latest

WORKDIR /app

# 安装 CA 证书
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 从构建阶段复制二进制文件
COPY --from=builder /app/tg-forward /app/tg-forward

# 创建配置和数据目录
RUN mkdir -p /app/config /app/data

# 复制配置文件
COPY config/config.yaml /app/config/

# 设置卷
VOLUME ["/app/config", "/app/data"]

# 设置入口点
ENTRYPOINT ["/app/tg-forward"]

# 默认命令
CMD ["-config", "/app/config/config.yaml"] 