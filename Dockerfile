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
# 第三阶段：最终生产镜像（整合前端 Nginx + 后端 Go）
# ==========================================
FROM alpine:3.18
RUN apk add --no-cache nginx supervisor tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

WORKDIR /app

# 复制后端可执行程序
COPY --from=backend-builder /app/backend/xray-monitor-backend /app/xray-monitor-backend

# 复制前端打包后的静态资源到 Nginx 默认目录
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html

# 覆盖 Nginx 配置文件（支持前端路由及 API 内部反代）
RUN echo ' \
server { \
    listen 10000; \
    server_name localhost; \
    \
    # 前端静态页面 \
    location / { \
        root /usr/share/nginx/html; \
        index index.html; \
        try_files $uri $uri/ /index.html; \
    } \
    \
    # 后端 API 反代 \
    location /api/ { \
        proxy_pass http://127.0.0.1:8080; \
        proxy_set_header Host $host; \
        proxy_set_header X-Real-IP $remote_addr; \
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; \
    } \
}' > /etc/nginx/http.d/default.conf

# 使用 Supervisor 同时守护 Go 后端和 Nginx 前端进程
RUN echo ' \
[supervisorctl] \
\
[supervisor] \
nodaemon=true \
user=root \
\
[program:backend] \
command=/app/xray-monitor-backend \
autostart=true \
autorestart=true \
stdout_logfile=/dev/stdout \
stdout_logfile_maxbytes=0 \
stderr_logfile=/dev/stderr \
stderr_logfile_maxbytes=0 \
\
[program:nginx] \
command=nginx -g "daemon off;" \
autostart=true \
autorestart=true \
stdout_logfile=/dev/stdout \
stdout_logfile_maxbytes=0 \
stderr_logfile=/dev/stderr \
stderr_logfile_maxbytes=0 \
' > /etc/supervisord.conf

# 创建可能需要的日志空目录（防御性）
RUN mkdir -p /var/lib/marzban

# 暴露容器内部的 10000 端口
EXPOSE 10000

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]