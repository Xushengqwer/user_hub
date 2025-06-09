# user_hub/Dockerfile (修正版)

# --- 构建阶段 (Builder Stage) ---
FROM golang:1.23-alpine AS builder
LABEL stage=builder

WORKDIR /app

# [优化] 先只复制 go.mod 和 go.sum 以下载依赖，充分利用缓存
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制所有源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o user_hub_app ./main.go

# --- 运行阶段 (Runtime Stage) ---
FROM alpine:latest

# 安装 ca-certificates 以支持 HTTPS 等外部连接
RUN apk add --no-cache ca-certificates

# 创建非 root 用户，增强安全性
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /app/user_hub_app .

# [核心修正] 1. 创建 config 目录
RUN mkdir -p /app/config
# [核心修正] 2. 将配置文件复制到正确的路径
COPY ./config/config.development.yaml /app/config/config.development.yaml

# 暴露容器端口
EXPOSE 8081

# 切换到非 root 用户运行
USER appuser

# [核心修正] 3. CMD 命令指向正确的配置文件路径
CMD ["./user_hub_app", "-config", "/app/config/config.development.yaml"]