# MediaWarp + FontInAss 容器化部署

本项目提供了将 MediaWarp 和 FontInAss 打包到同一个 Docker 容器中的解决方案，简化部署和管理。

## 快速开始

### 1. 构建镜像

```bash
# 使用构建脚本
./docker/build.sh

# 或直接使用 Docker Compose
docker-compose build
```

### 2. 运行容器

```bash
# 生产环境
docker-compose up -d

# 开发环境
docker-compose -f docker-compose.dev.yml up -d

# 查看日志
docker-compose logs -f
```

### 3. 访问服务

- **MediaWarp**: http://localhost:9000
- **FontInAss**: http://localhost:8011  # 仅在容器内部使用，不对外暴露

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `TZ` | Asia/Shanghai | 时区设置 |
| `MEDIAWARP_DEV` | false | 开发模式 |

### 端口映射

| 服务 | 容器端口 | 主机端口 | 说明 |
|------|----------|----------|------|
| MediaWarp | 9000 | 9000 | 媒体代理服务 |
| FontInAss | 8011 | - | 字幕处理服务（容器内部通信） |

### 配置文件

容器使用 `/config/config.yaml` 作为配置文件。可以通过挂载卷来自定义配置：

```yaml
volumes:
  - ./my-config:/config:ro
```

## 容器架构

### 服务关系

```
客户端请求 → MediaWarp (9000) → FontInAss (8011, 内部) → 返回处理结果
```

### 健康检查

容器包含健康检查机制，检查 MediaWarp 服务的可用性：

```bash
# 手动检查健康状态
docker inspect --format='{{.State.Health.Status}}' mediawarp-with-fontinass
```

## 部署选项

### 单容器部署（推荐）

使用 `docker-compose.yml` 文件，适合大多数场景：

```bash
docker-compose up -d
```

### 开发环境部署

使用 `docker-compose.dev.yml` 文件，支持热重载：

```bash
docker-compose -f docker-compose.dev.yml up -d
```

### Kubernetes 部署

可以创建 Kubernetes 部署文件：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mediawarp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mediawarp
  template:
    metadata:
      labels:
        app: mediawarp
    spec:
      containers:
      - name: mediawarp
        image: mediawarp-with-fontinass:latest
        ports:
        - containerPort: 9000
        # FontInAss 端口不需要暴露，容器内部通信
        volumeMounts:
        - name: config
          mountPath: /app/config
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: mediawarp-config
```

## 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 修改 docker-compose.yml 中的端口映射
   ports:
     - "9090:9000"  # 修改主机端口
   ```

2. **配置错误**
   ```bash
   # 检查配置文件语法
   docker-compose logs mediawarp | grep -i error
   ```

3. **服务启动失败**
   ```bash
   # 查看详细日志
   docker-compose logs --tail=100 mediawarp
   ```

### 日志查看

```bash
# 实时查看日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs mediawarp

# 查看最近错误
docker-compose logs --tail=50 | grep -i error
```

## 性能优化

### 资源限制

在生产环境中建议设置资源限制：

```yaml
services:
  mediawarp:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '1.0'
        reservations:
          memory: 256M
          cpus: '0.5'
```

### 缓存优化

根据实际使用情况调整缓存设置：

```yaml
cache:
  enable: true
  image_ttl: 24h      # 图片缓存时间
  subtitle_ttl: 12h   # 字幕缓存时间
```

## 安全考虑

1. **使用非 root 用户运行**
2. **只读挂载配置文件**
3. **限制容器网络访问**
4. **定期更新基础镜像**

## 更新和维护

### 更新镜像

```bash
# 拉取最新代码
git pull

# 重新构建镜像
docker-compose build --no-cache

# 重启服务
docker-compose down
docker-compose up -d
```

### 数据备份

```bash
# 备份配置文件
tar -czf mediawarp-config-backup.tar.gz config/

# 备份日志文件
tar -czf mediawarp-logs-backup.tar.gz logs/
```

### 路径配置说明

容器内的路径配置已标准化为：

- **应用目录**: `/app` (MediaWarp 和 FontInAss 程序文件)
- **配置文件路径**: `/app/config/config.yaml`
- **日志目录**: `/app/logs`

这种配置与 MediaWarp 的默认路径一致，无需额外的环境变量配置。