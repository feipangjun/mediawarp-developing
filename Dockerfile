# MediaWarp + FontInAss 统一容器镜像
# 多阶段构建，将 MediaWarp (Go) 和 FontInAss (Python) 打包到同一个容器中

# 第一阶段：构建 MediaWarp (Go)
FROM golang:1.26-alpine AS mediawarp-builder

# 设置构建参数
ARG APP_VERSION=1.0.1dev
ARG COMMIT_HASH=ea77acc574ddd5eee477b6c091fc9f03be6bb788
ARG BUILD_DATE=2026-03-24

# 安装构建依赖
RUN apk add --no-cache git build-base

# 设置工作目录
WORKDIR /build

# 复制 Go 模块文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建 MediaWarp
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "\
    -s -w \
    -X MediaWarp/internal/config.appVersion=${APP_VERSION} \
    -X MediaWarp/internal/config.commitHash=${COMMIT_HASH} \
    -X MediaWarp/internal/config.buildDate=${BUILD_DATE}" \
    -o MediaWarp main.go

# 第二阶段：构建 FontInAss (Python)
FROM python:3.10-alpine AS fontinass-builder

# 设置工作目录
WORKDIR /build/fontinass

# 复制 FontInAss 源代码
COPY fontinass/ .

# 安装 Python 依赖
RUN pip install --no-cache-dir -r requirements.txt

# 第三阶段：最终镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建应用用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从第一阶段复制 MediaWarp 二进制文件
COPY --from=mediawarp-builder /build/MediaWarp ./

# 从第二阶段复制 FontInAss
COPY --from=fontinass-builder /build/fontinass ./fontinass/

# 复制默认配置文件
COPY config/config.yaml.example ./config/config.yaml

# 创建必要的目录
RUN mkdir -p /logs && chown -R appuser:appgroup /logs

# 启动脚本
COPY docker/start.sh ./start.sh

# 设置文件权限（在切换用户之前）
RUN chmod +x /app/MediaWarp && chmod +x ./start.sh

# 切换用户
USER appuser

# MediaWarp暴露端口
EXPOSE 9000

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9000/health || exit 1

# 启动服务
CMD ["./start.sh"]