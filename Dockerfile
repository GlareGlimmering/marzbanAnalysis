# ==========================================
# 第一阶段：前端 Vue 生产环境打包
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ==========================================
# 第二阶段：后端 Go 静态编译
# ==========================================
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app/backend
# 启用 CGO 是因为 SQLite 驱动需要
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 静态编译 Go 二进制
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o xray-monitor-backend main.go

# ==========================================
# 第三阶段：最终生产镜像（纯净 Go 驱动版本，告别 Nginx/Supervisor 冲突）
# ==========================================
FROM alpine:3.18
RUN apk add --no-cache tzdata ca-certificates

# 设置时区为上海
ENV TZ=Asia/Shanghai

WORKDIR /app

# 1. 复制后端可执行程序到当前工作目录
COPY --from=backend-builder /app/backend/xray-monitor-backend /app/xray-monitor-backend

# 2. 🔥 核心修正：直接把前端编译产物复制到 Go 能够通过 "./dist" 相对路径访问的地方
COPY --from=frontend-builder /app/frontend/dist /app/dist

# 创建需要的日志空目录
RUN mkdir -p /var/lib/marzban

# 暴露容器内部的 10000 端口
EXPOSE 10000

# 直接启动 Go 后端，不再通过 supervisor 转手
CMD ["/app/xray-monitor-backend"]