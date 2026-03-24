# GitHub Secrets 配置指南

本文档详细说明如何在 GitHub 仓库中配置 Docker Hub 访问令牌。

## 🔑 配置 GitHub Secrets 的步骤

### 第一步：获取 Docker Hub 访问令牌

1. **登录 Docker Hub**
   - 访问 [Docker Hub 网站](https://hub.docker.com)
   - 使用你的账号登录（用户名：sadnano）

2. **创建访问令牌**
   - 点击右上角头像 → **Account Settings**
   - 左侧菜单选择 **Security**
   - 点击 **Access Tokens**
   - 点击 **New Access Token**

3. **设置令牌信息**
   - **Description**: `GitHub Actions for mediawarp-with-fontinass`
   - **Access type**: 选择 **Read & Write**（需要推送镜像的权限）
   - 点击 **Generate**

4. **复制令牌**
   - **重要**：立即复制生成的令牌值
   - 令牌只会显示一次，请妥善保存

### 第二步：在 GitHub 仓库中添加 Secrets

1. **访问 GitHub 仓库**
   - 打开 [feipangjun/mediawarp-developing](https://github.com/feipangjun/mediawarp-developing)

2. **进入仓库设置**
   - 点击仓库顶部的 **Settings** 标签
   - 左侧菜单选择 **Secrets and variables** → **Actions**

3. **添加 Secrets**
   - 点击 **New repository secret**
   - 添加以下两个 Secrets：

#### Secret 1: DOCKERHUB_USERNAME
- **Name**: `DOCKERHUB_USERNAME`
- **Secret**: `sadnano`

#### Secret 2: DOCKERHUB_TOKEN
- **Name**: `DOCKERHUB_TOKEN`
- **Secret**: 粘贴你从 Docker Hub 复制的令牌值

4. **保存设置**
   - 点击 **Add secret** 保存每个 Secret

### 第三步：验证配置

1. **检查 GitHub Actions**
   - 访问仓库的 **Actions** 标签
   - 查看 "Build and Publish Docker Image" 工作流
   - 如果之前构建失败，现在应该可以重新运行

2. **手动触发构建**
   - 在 Actions 页面，找到 "Build and Publish Docker Image"
   - 点击 **Run workflow**
   - 选择分支：`main`
   - 点击 **Run workflow**

## 🔍 验证构建成功

构建完成后，你可以通过以下方式验证：

### 1. 检查 GitHub Actions 日志
- 查看工作流运行详情
- 确认所有步骤都成功完成

### 2. 检查 Docker Hub
- 访问 [Docker Hub 仓库](https://hub.docker.com/r/sadnano/mediawarp-with-fontinass)
- 确认有新版本的镜像

### 3. 测试拉取镜像
```bash
# 拉取最新镜像
docker pull sadnano/mediawarp-with-fontinass:latest

# 查看镜像信息
docker images sadnano/mediawarp-with-fontinass

# 测试运行
docker run --rm sadnano/mediawarp-with-fontinass:latest --version
```

## 🛠️ 故障排除

### 常见问题

1. **"unauthorized: authentication required"**
   - 检查 DOCKERHUB_TOKEN 是否正确
   - 确认令牌有读写权限

2. **构建失败**
   - 检查 Dockerfile 语法
   - 查看构建日志中的具体错误

3. **推送失败**
   - 确认 DOCKERHUB_USERNAME 和 TOKEN 都正确配置
   - 检查网络连接

### 重新生成令牌

如果令牌丢失或泄露：
1. 在 Docker Hub 中删除旧令牌
2. 生成新令牌
3. 在 GitHub 中更新 DOCKERHUB_TOKEN Secret

## 📞 支持

如果遇到问题：
1. 查看 GitHub Actions 详细日志
2. 检查 Docker Hub 账户权限
3. 在项目 Issues 中报告问题

## 🔗 相关链接

- [GitHub Secrets 文档](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Docker Hub Access Tokens](https://docs.docker.com/docker-hub/access-tokens/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)