# 阶段 1：构建阶段
FROM golang:1.23 AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件，用于下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制整个项目源码
COPY . .

# 编译 Go 程序，生成可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -o user_hub main.go

# 阶段 2：运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的可执行文件
COPY --from=builder /app/user_hub .

# 暴露端口（根据你的项目调整，例如 8080）
EXPOSE 8080

# 运行程序
CMD ["./user_hub"]