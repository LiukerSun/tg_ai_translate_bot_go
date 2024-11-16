# 构建阶段
FROM golang:1.22-bullseye AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tg-bot-go main.go

# 运行阶段
FROM alpine:latest

# 设置时区
ENV TZ=Asia/Shanghai

# 安装必要的包
RUN apk update && \
    apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates

# 创建必要的目录
RUN mkdir -p /app/logs

WORKDIR /app

# 从构建阶段复制文件
COPY --from=builder /app/tg-bot-go .
COPY --from=builder /app/config ./config

# 设置权限
RUN chmod +x /app/tg-bot-go


# 运行程序
CMD ["./tg-bot-go"]