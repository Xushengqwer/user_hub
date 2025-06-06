# user_hub/Dockerfile

# --- 构建阶段 (Builder Stage) ---
FROM golang:1.23-alpine AS builder
LABEL stage=builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o user_hub_app ./main.go

# --- 运行阶段 (Runtime Stage) ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/user_hub_app .

# 明确地将开发配置文件复制到镜像中作为备用/基础配置
COPY ./config/config.development.yaml /app/config.development.yaml

EXPOSE 8081

USER appuser
# CMD 明确指向我们复制进来的配置文件路径
CMD ["./user_hub_app", "-config", "/app/config.development.yaml"]