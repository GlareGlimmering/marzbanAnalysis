#!/bin/bash

# 字体颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}===============================================${NC}"
echo -e "${GREEN}    XRAY-MONITOR 一键生产环境部署脚本 (Tunnel版) ${NC}"
echo -e "${GREEN}===============================================${NC}"

# 1. 检查是否为 Root 用户
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}错误：请使用 root 用户或 sudo 运行此脚本！${NC}"
  exit 1
fi

# 2. 检查并安装 Docker / Docker Compose
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}[1/3] 未检测到 Docker，正在全力安装中...${NC}"
    apt-get update && apt-get install -y curl
    curl -fsSL https://get.docker.com | sh
    systemctl start docker
    systemctl enable docker
else
    echo -e "${GREEN}[1/3] Docker 环境检测正常。${NC}"
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}[2/3] 未检测到 Docker Compose，正在安装...${NC}"
    apt-get install -y docker-compose
else
    echo -e "${GREEN}[2/3] Docker Compose 检测正常。${NC}"
fi

# 3. 校验日志文件是否存在，防止 Docker 挂载成文件夹错误
LOG_PATH="/var/lib/marzban/xray_access.log"
if [ ! -f "$LOG_PATH" ]; then
    echo -e "${YELLOW}警告：检测到当前路径下不存在 $LOG_PATH${NC}"
    echo -e "${YELLOW}正在自动为你创建一个空的日志占位符，以防挂载失败...${NC}"
    mkdir -p /var/lib/marzban
    touch "$LOG_PATH"
fi

# 4. 执行 Docker Compose 编译与拉起
echo -e "${GREEN}[3/3] 开始构建镜像并启动 XRAY-MONITOR 容器...${NC}"
docker-compose up -d --build

# 5. 部署状态检查与输出
if [ $? -eq 0 ]; then
    echo -e "${GREEN}===============================================${NC}"
    echo -e "${GREEN}🎉 部署成功！XRAY-MONITOR 正在平稳运行。${NC}"
    echo -e "${YELLOW}🔒 安全提示：服务已死死锁定在内网本地端口：${NC}"
    echo -e "${GREEN}URL: http://127.0.0.1:10000${NC}"
    echo -e "${YELLOW}你现在可以在 Cloudflare Tunnel 后台将此端口映射出去。${NC}"
    echo -e "${GREEN}===============================================${NC}"
else
    echo -e "${RED}❌ 抱歉，容器构建或启动过程中发生致命错误，请检查日志。${NC}"
fi