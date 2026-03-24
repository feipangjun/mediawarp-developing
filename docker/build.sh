#!/bin/bash

# MediaWarp + FontInAss 容器构建脚本

echo "开始构建 MediaWarp + FontInAss 容器..."

# 检查 Docker 是否可用
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装或未在 PATH 中"
    exit 1
fi

# 构建镜像
echo "构建 Docker 镜像..."
docker build -t mediawarp-with-fontinass:latest .

if [ $? -eq 0 ]; then
    echo "镜像构建成功"
    echo "镜像标签: mediawarp-with-fontinass:latest"
else
    echo "镜像构建失败"
    exit 1
fi

# 可选：推送镜像到镜像仓库
if [ "$1" = "--push" ]; then
    echo "推送镜像到镜像仓库..."
    # 这里可以添加推送逻辑
    echo "推送功能待实现"
fi

echo "构建完成！"
echo ""
echo "运行容器:"
echo "  docker-compose up -d"
echo ""
echo "查看日志:"
echo "  docker-compose logs -f"
echo ""
echo "停止容器:"
echo "  docker-compose down"