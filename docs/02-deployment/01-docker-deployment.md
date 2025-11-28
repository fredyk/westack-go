# Docker Deployment Guide

## Overview

westack-go v2 includes a production-ready Dockerfile that creates a secure, minimal Docker image for deploying your API applications.

## Image Characteristics

### Multi-Stage Build

The Dockerfile uses a two-stage build process:

1. **Builder Stage**: Compiles the Go application
2. **Runtime Stage**: Creates minimal image with only the binary

**Benefits**:
- Smaller final image size
- No build tools in production image
- Reduced attack surface

### Security Features

- ✅ **Non-root User**: Runs as `westack` user (not root)
- ✅ **Static Binary**: CGO_ENABLED=0 for no external dependencies
- ✅ **Minimal Base**: Uses Alpine Linux (small, secure)
- ✅ **SSL Certificates**: Includes CA certificates for HTTPS
- ✅ **Timezone Data**: Includes timezone information

### Health Checking

The image includes a built-in health check:
- **Endpoint**: `http://localhost:8023/health`
- **Interval**: Every 5 seconds
- **Timeout**: 3 seconds

## Building the Image

### Basic Build

```bash
cd v2
docker build -t myapp:latest .
```

### With Custom Go Version

```bash
docker build \
  --build-arg GO_VERSION=1.23 \
  -t myapp:latest \
  .
```

### With Custom User

```bash
docker build \
  --build-arg USER=myappuser \
  -t myapp:latest \
  .
```

### Multi-platform Build

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t myapp:latest \
  --push .
```

## Running the Container

### Basic Run

```bash
docker run -d \
  --name myapp \
  -p 8023:8023 \
  myapp:latest
```

### With Environment Variables

```bash
docker run -d \
  --name myapp \
  -p 8023:8023 \
  -e JWT_SECRET=your-secret-key \
  -e DATABASE_URL=mongodb://host:port/db \
  myapp:latest
```

### With Volume Mounts

```bash
docker run -d \
  --name myapp \
  -p 8023:8023 \
  -v /path/to/config:/home/westack/config:ro \
  myapp:latest
```

### With Health Check Monitoring

```bash
docker run -d \
  --name myapp \
  -p 8023:8023 \
  --health-cmd "wget -q http://localhost:8023/health || exit 1" \
  --health-interval=5s \
  --health-timeout=3s \
  --health-retries=3 \
  myapp:latest
```

## Docker Compose

Example `docker-compose.yml`:

```yaml
version: '3.8'

services:
  westack-app:
    build:
      context: ./v2
      args:
        GO_VERSION: "1.22"
        USER: westack
    ports:
      - "8023:8023"
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - DATABASE_URL=${DATABASE_URL}
    volumes:
      - ./config:/home/westack/config:ro
    healthcheck:
      test: ["CMD", "wget", "-q", "http://localhost:8023/health"]
      interval: 5s
      timeout: 3s
      retries: 3
    restart: unless-stopped
```

## Production Considerations

### Security

1. **Never run as root**: The image already uses a non-root user
2. **Use secrets management**: Don't hardcode sensitive values
3. **Enable health checks**: Monitor container health
4. **Limit resources**: Set memory and CPU limits

```bash
docker run -d \
  --name myapp \
  -p 8023:8023 \
  --memory="512m" \
  --cpus="1.0" \
  --read-only \
  --tmpfs /tmp \
  myapp:latest
```

### Networking

For production, use:
- Reverse proxy (nginx, traefik)
- HTTPS termination
- Network isolation

### Monitoring

Monitor:
- Health check status: `docker inspect --format='{{.State.Health.Status}}' myapp`
- Container logs: `docker logs -f myapp`
- Resource usage: `docker stats myapp`

## Troubleshooting

### Container Won't Start

Check logs:
```bash
docker logs myapp
```

Common issues:
- Missing environment variables
- Port already in use
- Health check failing

### Health Check Failing

Verify endpoint:
```bash
docker exec myapp wget -q -O- http://localhost:8023/health
```

### Permission Issues

Ensure volumes have correct permissions for the `westack` user (UID typically 1000).

## CI/CD Integration

The project includes GitHub Actions workflow (`.github/workflows/build.yml`) that:
- Builds multi-platform images
- Pushes to Docker registry
- Uses build cache for efficiency

## Image Optimization

The current image is optimized for:
- Small size (Alpine Linux base)
- Fast startup (static binary)
- Security (non-root, minimal attack surface)

**Typical image size**: ~20-30 MB (depending on application)

## Related

- GitHub Actions Build Workflow
- Health Check Endpoints
- Environment Configuration
