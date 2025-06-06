# user_hub/Dockerfile

# 使用一个官方的 Go 镜像作为构建环境
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件，并下载依赖项
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制所有源代码
COPY . .

# 构建应用
# CGO_ENABLED=0 确保静态链接
# -ldflags="-w -s" 减小可执行文件大小 (可选)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o user_hub_app ./main.go

# --- 运行阶段 ---
# 使用一个轻量级的基础镜像
FROM alpine:latest

# 添加必要的 CA 证书，如果应用需要进行 HTTPS 调用
RUN apk add --no-cache ca-certificates

WORKDIR /app

# 从构建阶段复制编译好的应用
COPY --from=builder /app/user_hub_app .

# (重要) 将配置文件从构建上下文复制到容器中应用期望的位置
# 这里假设您的应用会从 /app/config/config.development.yaml 读取
# 注意：如果您通过 docker-compose.yml 的 volumes 挂载了整个 config 目录，
# 这一步 COPY ./config /app/config 也可以，但 volumes 会覆盖它。
# 为了清晰，并且如果 volumes 挂载的是单个文件，这里最好还是创建目录结构。
RUN mkdir -p /app/config
# COPY ./config/config.development.yaml /app/config/config.development.yaml
# 如果您在 docker-compose.yml 中挂载整个 ./config 目录到 /app/config，则上面的 COPY 不是必需的，
# 但保留 RUN mkdir -p /app/config 是好的，以防万一。

# 暴露应用监听的端口
EXPOSE 8081

# 运行应用，并指定配置文件路径
# 这个路径需要与 docker-compose.yml 中挂载的配置目标路径（如果是文件挂载）
# 或应用内部期望的路径（如果是目录挂载后，应用自行拼接文件名）一致。
CMD ["./user_hub_app", "-config", "/app/config/config.development.yaml"]