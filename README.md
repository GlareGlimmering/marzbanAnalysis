# XRAY-MONITOR 🚀

一个轻量、硬核的 Xray/Marzban 行为日志实时透视大屏。支持全局入站/出站热度分流饼图，多设备流量混淆排查工具。

由 Go (Fiber) + Vue 3 (Vite + ECharts) 驱动，通过 GitHub Actions 自动编译，支持跨平台一键容器化私密部署。

基于Marzaban二次开发。

使用美国豆包+codex编译。

---
## 🛠️ 1. Marzban 面板配置 (数据源)

修改 Marzban 核心配置文件（通常位于 `/var/lib/marzban/xray_config.json`），将 `log` 字段严格调整为：

```json
"log": {
  "loglevel": "info",
  "access": "/var/lib/marzban/xray_access.log",
}
```

💡 提示：修改完成后，重启 Marzban 面板以激活日志输出。

## 🐳 2. 服务器一键部署指令
本镜像已完全容器化，且为了安全隐私，仅在服务器本地 (127.0.0.1) 暴露 10000 端口，公网无法直接扫描或访问。

在Debian/Ubuntu 服务器上，直接右手复制并执行以下“一行命令”即可完成全自动部署：

### 安装 Docker Compose 依赖
```
apt update && apt install docker-compose -y
```

### 创建并切入目录
```
mkdir -p /opt/xray-monitor && cd /opt/xray-monitor
```

### 一键下载配置并后台拉起大屏服务
```
cat << 'EOF' > docker-compose.yml
services:
  xray-monitor:
    image: ghcr.io/glareglimmering/marzbananalysis:latest
    container_name: xray-monitor
    restart: always
    ports:
      - "127.0.0.1:10000:10000"
    volumes:
      - /var/lib/marzban/xray_access.log:/var/lib/marzban/xray_access.log:ro
      - ./data:/app/store/data
    environment:
      - TZ=Asia/Shanghai
EOF
```

### 拉起服务
```
docker compose up -d 2>/dev/null || docker-compose up -d
```

## 🔒 3. Cloudflare Tunnel 私密访问配置
推荐使用 Cloudflare Tunnel 进行无公网端口暴露的安全穿透与鉴权访问。

强烈推荐开启零信任模式。

---

## 其他指令

**查看运行状态**  
```
cd /opt/xray-monitor
docker ps | grep xray-monitor
```

**重启服务**
```
cd /opt/xray-monitor
docker compose restart || docker-compose restart
```

**停止并关闭**
```
cd /opt/xray-monitor
docker compose down || docker-compose down
```

**升级到最新版**
```
cd /opt/xray-monitor
docker compose pull && docker compose up -d || docker-compose pull && docker-compose up -d
```

## 备注
每次更新至最新版本后，项目会重新读取日志文件，如何更新而不覆盖源数据库，还在和美国豆包商议中。