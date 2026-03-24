# Docker Hub 发布指南

本文档介绍如何将 MediaWarp + FontInAss 容器镜像发布到 Docker Hub。

## 📦 镜像信息

- **Docker Hub 用户名**: `sadnano`
- **镜像名称**: `mediawarp-with-fontinass`
- **完整镜像地址**: `sadnano/mediawarp-with-fontinass`

## 🚀 快速开始

### 从 Docker Hub 拉取镜像

```bash
# 拉取最新版本
docker pull sadnano/mediawarp-with-fontinass:latest

# 拉取特定版本
docker pull sadnano/mediawarp-with-fontinass:v1.0.0
```

### 运行容器

```bash
# 使用 Docker Compose
docker-compose up -d

# 直接运行
docker run -d \
  --name mediawarp-with-fontinass \
  -p 9000:9000 \
  -v ./config:/app/config \
  -v ./logs:/app/logs \
  sadnano/mediawarp-with-fontinass:latest
```

## 🔧 手动发布流程

### 1. 准备工作

确保服务器已安装 Docker 并登录 Docker Hub：

```bash
# 登录 Docker Hub
docker login -u sadnano

# 输入密码或访问令牌
```

### 2. 使用发布脚本

项目提供了自动发布脚本：

```bash
# 进入项目根目录
cd /path/to/MediaWarp-main

# 运行发布脚本
./docker/publish.sh
```

发布脚本支持以下选项：
- **单架构构建** (默认)
- **多架构构建** (amd64 + arm64)
- **仅构建不推送**

### 3. 手动构建命令

如果需要手动控制构建过程：

```bash
# 构建镜像
docker build -t sadnano/mediawarp-with-fontinass:latest .

# 推送镜像
docker push sadnano/mediawarp-with-fontinass:latest

# 多架构构建 (需要 docker buildx)
docker buildx build --platform linux/amd64,linux/arm64 \
  -t sadnano/mediawarp-with-fontinass:latest \
  --push .
```

## 🤖 自动化发布

项目配置了 GitHub Actions，在以下情况下自动发布：

1. **推送标签时** (如 `v1.0.0`)
2. **手动触发** (GitHub Actions 界面)

### GitHub Secrets 配置

在 GitHub 仓库设置中添加以下 Secrets：

- `DOCKERHUB_USERNAME`: sadnano
- `DOCKERHUB_TOKEN`: Docker Hub 访问令牌

## 🏷️ 版本标签策略

| 标签 | 说明 | 示例 |
|------|------|------|
| `latest` | 最新稳定版本 | `sadnano/mediawarp-with-fontinass:latest` |
| `v1.0.0` | 语义化版本 | `sadnano/mediawarp-with-fontinass:v1.0.0` |
| `v1.0` | 主次版本 | `sadnano/mediawarp-with-fontinass:v1.0` |
| `git-sha` | Git 提交哈希 | `sadnano/mediawarp-with-fontinass:a1b2c3d` |

## 📊 镜像信息

### 镜像大小
- **压缩后**: ~150MB
- **解压后**: ~400MB

### 支持的架构
- `linux/amd64` (Intel/AMD 64位)
- `linux/arm64` (ARM 64位)

### 包含的服务
- **MediaWarp**: Go 应用，端口 9000
- **FontInAss**: Python 应用，端口 8011 (内部)

## 🔍 验证发布

发布后可以通过以下方式验证：

```bash
# 查看镜像信息
docker images sadnano/mediawarp-with-fontinass

# 测试运行
docker run --rm sadnano/mediawarp-with-fontinass:latest --version

# 检查 Docker Hub 页面
open https://hub.docker.com/r/sadnano/mediawarp-with-fontinass
```

## 🛠️ 故障排除

### 常见问题

1. **权限错误**
   ```bash
   # 确保已登录 Docker Hub
   docker login -u sadnano
   ```

2. **构建失败**
   ```bash
   # 清理构建缓存
   docker system prune -a
   
   # 重新构建
   docker build --no-cache -t sadnano/mediawarp-with-fontinass:latest .
   ```

3. **多架构构建失败**
   ```bash
   # 启用 buildx
   docker buildx create --name multiarch --use
   
   # 重新构建
   docker buildx build --platform linux/amd64,linux/arm64 --push .
   ```

## 📞 支持

如果遇到问题，请：

1. 检查 Docker 日志：`docker logs <container_id>`
2. 查看 GitHub Actions 运行日志
3. 在项目 Issues 中报告问题

## 🔗 相关链接

- [Docker Hub 仓库](https://hub.docker.com/r/sadnano/mediawarp-with-fontinass)
- [GitHub 项目](https://github.com/feipangjun/mediawarp-developing)
- [Docker 文档](https://docs.docker.com/)