---
name: "docker-image-builder"
description: "Builds and publishes Docker images to Docker Hub. Invoke when user wants to package and distribute container images."
---

# Docker Image Builder

This skill helps build Docker images and publish them to Docker Hub or other container registries.

## When to Use

- User wants to build a Docker image from a Dockerfile
- User wants to publish an image to Docker Hub
- User needs to tag and version Docker images
- User wants to automate the build and publish process

## Key Features

1. **Multi-architecture builds**: Support for different CPU architectures
2. **Version tagging**: Automatic version management
3. **Build optimization**: Multi-stage builds and caching
4. **Registry integration**: Docker Hub, GitHub Container Registry, etc.

## Usage Examples

- Build and push to Docker Hub: `docker build -t username/repo:tag . && docker push username/repo:tag`
- Multi-architecture builds: `docker buildx build --platform linux/amd64,linux/arm64 -t username/repo:tag . --push`
- Automated builds with GitHub Actions

## Best Practices

- Use semantic versioning for tags
- Include `latest` tag for the most recent stable version
- Use multi-stage builds for smaller images
- Sign images for security verification
- Use build arguments for configuration