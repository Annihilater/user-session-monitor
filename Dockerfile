# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git make

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 go build -o user-session-monitor ./cmd/monitor

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN adduser -D -u 1000 monitor

# 复制二进制文件
COPY --from=builder /app/user-session-monitor .

# 设置权限
RUN chown -R monitor:monitor /app

# 切换到非 root 用户
USER monitor

# 运行应用
CMD ["./user-session-monitor"] 