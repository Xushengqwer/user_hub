# 阶段 1：构建阶段
# 使用官方 Golang 1.23 镜像作为构建基础镜像
FROM golang:1.23 AS builder

# 设置工作目录为 /app，所有后续命令都在这个目录下执行
WORKDIR /app

# 复制 go.mod 和 go.sum 文件到工作目录，用于管理项目依赖
COPY go.mod go.sum ./

# 下载所有依赖包到本地，确保构建过程的依赖完整性
RUN go mod download

# 将当前目录下的所有源代码文件复制到容器的工作目录
COPY . .

# 编译 Go 程序：
# - CGO_ENABLED=0：禁用 CGO，避免引入 C 语言依赖，提高跨平台兼容性
# - GOOS=linux：指定目标操作系统为 Linux，确保在容器中运行正常
# - go build -o user_hub main.go：将 main.go 编译为可执行文件 user_hub
RUN CGO_ENABLED=0 GOOS=linux go build -o user_hub main.go

# 阶段 2：运行阶段
# 使用轻量级的 Alpine Linux 镜像作为运行时基础镜像，减小镜像体积
FROM alpine:latest

# 设置工作目录为 /app，运行时的命令在此目录执行
WORKDIR /app

# 从构建阶段（builder）中复制编译好的可执行文件 user_hub 到当前镜像
COPY --from=builder /app/user_hub .

# 复制配置文件目录 common/config/ 到镜像的相同路径
# 确保应用运行时可以读取到必要的配置文件
COPY common/config/ ./common/config/

# 声明容器监听的端口为 80
# 与微信云托管的要求保持一致（通常要求 80 端口提供 HTTP 服务）
EXPOSE 80

# 设置容器启动时执行的默认命令
# 运行编译好的 user_hub 可执行文件，启动应用程序
CMD ["./user_hub"]