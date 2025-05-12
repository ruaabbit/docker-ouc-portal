# 使用官方 Golang 镜像作为构建环境
FROM golang:1.20-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 Go 模块文件
COPY go.mod ./
RUN go mod tidy

# 下载依赖
RUN go mod download

# 复制源代码到容器中
COPY main.go ./

# 构建 Go 应用
# CGO_ENABLED=0 禁用 CGO, 使得构建的二进制文件是静态链接的，更容易在 alpine 这种轻量级镜像中运行
# -ldflags="-s -w" 减小二进制文件大小 (去除符号表和调试信息)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /docker-ouc-portal main.go

# 使用一个轻量级的 Alpine 镜像作为最终的运行环境
FROM alpine:latest

# 设置时区
RUN apk --no-cache add tzdata

# 程序需要访问 CA 证书 (例如，进行 HTTPS 请求到外部服务)
RUN apk --no-cache add ca-certificates && update-ca-certificates

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /docker-ouc-portal .

# 暴露程序可能使用的端口
# EXPOSE 8080

# 设置默认的环境变量 (用户可以在 docker run 时覆盖)
ENV TZ="Asia/Shanghai"
ENV WLJF_USERNAME=""
ENV WLJF_PASSWORD=""
ENV WLJF_MODE="XHA"
ENV CHECK_INTERVAL_SECONDS="60"
ENV CHECK_TARGET_HOST="www.baidu.com:80"

# 容器启动时运行的命令
CMD ["./docker-ouc-portal"]
