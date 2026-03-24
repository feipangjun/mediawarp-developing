#!/bin/bash

# MediaWarp + FontInAss Docker 镜像发布脚本
# 用于构建并推送到 Docker Hub

set -e

# 配置变量
DOCKER_USERNAME="sadnano"
IMAGE_NAME="mediawarp-with-fontinass"
FULL_IMAGE_NAME="${DOCKER_USERNAME}/${IMAGE_NAME}"

# 版本信息（可以从 git 获取）
GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    # 检查是否已登录 Docker Hub
    if ! docker info | grep -q "Username: ${DOCKER_USERNAME}"; then
        log_warn "未检测到 Docker Hub 登录，请先运行: docker login"
        log_info "或者使用: docker login -u ${DOCKER_USERNAME}"
    fi
}

# 构建镜像
build_image() {
    local tag=$1
    local build_args=$2
    
    log_info "构建镜像: ${FULL_IMAGE_NAME}:${tag}"
    
    docker build \
        --build-arg APP_VERSION="${GIT_TAG}" \
        --build-arg COMMIT_HASH="${GIT_COMMIT}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        -t "${FULL_IMAGE_NAME}:${tag}" \
        .
    
    if [ $? -eq 0 ]; then
        log_info "镜像构建成功: ${FULL_IMAGE_NAME}:${tag}"
    else
        log_error "镜像构建失败"
        exit 1
    fi
}

# 推送镜像
push_image() {
    local tag=$1
    
    log_info "推送镜像到 Docker Hub: ${FULL_IMAGE_NAME}:${tag}"
    
    docker push "${FULL_IMAGE_NAME}:${tag}"
    
    if [ $? -eq 0 ]; then
        log_info "镜像推送成功: ${FULL_IMAGE_NAME}:${tag}"
    else
        log_error "镜像推送失败"
        exit 1
    fi
}

# 多架构构建（如果支持）
build_multi_arch() {
    if command -v docker buildx &> /dev/null; then
        log_info "检测到 docker buildx，启用多架构构建"
        
        # 创建并启用 buildx 构建器
        docker buildx create --name multiarch --use 2>/dev/null || true
        docker buildx use multiarch
        
        # 构建并推送多架构镜像
        docker buildx build \
            --platform linux/amd64,linux/arm64 \
            --build-arg APP_VERSION="${GIT_TAG}" \
            --build-arg COMMIT_HASH="${GIT_COMMIT}" \
            --build-arg BUILD_DATE="${BUILD_DATE}" \
            -t "${FULL_IMAGE_NAME}:${GIT_TAG}" \
            -t "${FULL_IMAGE_NAME}:latest" \
            --push \
            .
            
        if [ $? -eq 0 ]; then
            log_info "多架构镜像构建并推送成功"
        else
            log_warn "多架构构建失败，回退到单架构构建"
            build_and_push_single
        fi
    else
        log_warn "未检测到 docker buildx，使用单架构构建"
        build_and_push_single
    fi
}

# 单架构构建和推送
build_and_push_single() {
    # 构建版本标签
    build_image "${GIT_TAG}"
    
    # 构建 latest 标签
    build_image "latest"
    
    # 推送版本标签
    push_image "${GIT_TAG}"
    
    # 推送 latest 标签
    push_image "latest"
}

# 显示镜像信息
show_image_info() {
    log_info "=== 镜像信息 ==="
    log_info "镜像名称: ${FULL_IMAGE_NAME}"
    log_info "版本标签: ${GIT_TAG}"
    log_info "Git Commit: ${GIT_COMMIT}"
    log_info "构建时间: ${BUILD_DATE}"
    echo
}

# 主函数
main() {
    echo "=== MediaWarp + FontInAss Docker 镜像发布 ==="
    echo
    
    # 显示信息
    show_image_info
    
    # 检查 Docker
    check_docker
    
    # 询问构建方式
    echo "选择构建方式:"
    echo "1) 单架构构建 (默认)"
    echo "2) 多架构构建 (amd64 + arm64)"
    echo "3) 仅构建不推送"
    echo
    read -p "请输入选择 [1-3] (默认 1): " choice
    
    case "${choice:-1}" in
        1)
            log_info "开始单架构构建..."
            build_and_push_single
            ;;
        2)
            log_info "开始多架构构建..."
            build_multi_arch
            ;;
        3)
            log_info "仅构建镜像，不推送..."
            build_image "${GIT_TAG}"
            build_image "latest"
            ;;
        *)
            log_error "无效选择"
            exit 1
            ;;
    esac
    
    log_info "=== 发布完成 ==="
    echo
    log_info "镜像地址: https://hub.docker.com/r/${FULL_IMAGE_NAME}"
    log_info "拉取命令: docker pull ${FULL_IMAGE_NAME}:latest"
    echo
}

# 执行主函数
main "$@"