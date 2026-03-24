#!/bin/sh

# MediaWarp + FontInAss 统一启动脚本

# 设置环境变量
export PYTHONPATH=/app/fontinass/src:$PYTHONPATH

# 启动 FontInAss 服务（后台运行）
echo "启动 FontInAss 服务..."
cd /fontinass
python3 start.py &
FONTINASS_PID=$!

# 等待 FontInAss 启动
echo "等待 FontInAss 服务启动..."
sleep 5

# 检查 FontInAss 是否正常启动
if ! kill -0 $FONTINASS_PID 2>/dev/null; then
    echo "FontInAss 服务启动失败"
    exit 1
fi

# 启动 MediaWarp 服务（前台运行）
echo "启动 MediaWarp 服务..."
cd /mediawarp
./MediaWarp &
MEDIAWARP_PID=$!

# 等待 MediaWarp 启动
echo "等待 MediaWarp 服务启动..."
sleep 3

# 检查 MediaWarp 是否正常启动
if ! kill -0 $MEDIAWARP_PID 2>/dev/null; then
    echo "MediaWarp 服务启动失败"
    kill $FONTINASS_PID 2>/dev/null
    exit 1
fi

echo "所有服务已启动完成"
echo "- MediaWarp 运行在端口 9000 (外部可访问)"
echo "- FontInAss 运行在端口 8011 (容器内部通信)"

# 等待进程退出
wait $MEDIAWARP_PID
MEDIAWARP_EXIT=$?

# 停止 FontInAss
kill $FONTINASS_PID 2>/dev/null
wait $FONTINASS_PID 2>/dev/null

echo "服务已停止"
exit $MEDIAWARP_EXIT