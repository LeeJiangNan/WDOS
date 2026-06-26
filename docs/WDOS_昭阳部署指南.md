# WDOS 昭阳部署指南

> **环境**：昭阳笔记本 (Windows + WSL2 Ubuntu 24.04)
> **用途**：开发调试、内部测试
> **更新**：2026-06-24

---

## 一、环境概览

```
┌─ Windows ────────────────────────────────────────┐
│  端口转发: 9090/3000/5173 → WSL2                 │
│                                                   │
│  ┌─ WSL2 Ubuntu 24.04 ──────────────────────────┐│
│  │  Docker: mysql + redis + minio                ││
│  │  Go 后端: /home/corerain/WDOS/wdos-server     ││
│  │  Flutter: /tmp/wdos_web/web (python http)     ││
│  │  Vue:    /home/corerain/web_dist (python http)││
│  └──────────────────────────────────────────────┘│
└───────────────────────────────────────────────────┘
```

| 服务 | 端口 | 访问地址 |
|------|------|---------|
| API 后端 | 9090 | http://100.107.124.26:9090 |
| Flutter APP | 3000 | http://100.107.124.26:3000 |
| Vue Web 管理 | 5173 | http://100.107.124.26:5173 |
| MinIO 控制台 | 9001 | http://100.107.124.26:9001 |

---

## 二、首次部署

### 1. 安装 WSL2 + Ubuntu

```powershell
# 管理员 PowerShell
wsl --install -d Ubuntu-24.04
```

重启后进 Ubuntu，设置用户名密码。

### 2. 安装 Docker

```bash
# 在 WSL2 内
curl -fsSL https://get.docker.com | sudo bash
sudo usermod -aG docker $USER
# 退出重进
```

### 3. 启动基础服务

```bash
cd /home/corerain
mkdir -p WDOS/miniapp/build
```

创建 `docker-compose.yml`：

```yaml
version: '3'
services:
  mysql:
    image: mysql:8.0
    container_name: wdos-mysql
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: wdos_db
    ports:
      - "3306:3306"
    volumes:
      - ./mysql_data:/var/lib/mysql
    restart: always

  redis:
    image: redis:7-alpine
    container_name: wdos-redis
    ports:
      - "6379:6379"
    restart: always

  minio:
    image: minio/minio
    container_name: wdos-minio
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - ./minio_data:/data
    restart: always
```

```bash
docker compose up -d
# 创建 MinIO bucket
docker exec wdos-minio mc alias set local http://localhost:9000 minioadmin minioadmin
docker exec wdos-minio mc mb local/wdos
```

### 4. 部署应用

```bash
# 从 Mac 把文件传到昭阳
# 1) 本地编译
cd /path/to/WDOS
GOOS=linux GOARCH=amd64 go build -o wdos-server-linux ./cmd/api/
gzip wdos-server-linux

cd miniapp && flutter build web --no-tree-shake-icons
cd build && tar czf web.tar.gz web/

cd ../web && npm run build
cd dist && tar czf ../dist.tar.gz .

# 2) 上传到昭阳
scp wdos-server-linux.gz web.tar.gz dist.tar.gz chaoyang:C:/Users/corerain/

# 3) 在昭阳 WSL2 里执行
sudo pkill -9 -f wdos-server
cp /mnt/c/Users/corerain/wdos-server-linux.gz /home/corerain/WDOS/
cd /home/corerain/WDOS && gunzip -f wdos-server-linux.gz && chmod +x wdos-server-linux

# 启动后端
cd /home/corerain/WDOS
nohup ./wdos-server-linux > wdos.log 2>&1 &

# 部署 Flutter 前端
rm -rf /tmp/wdos_web && mkdir -p /tmp/wdos_web
cd /tmp/wdos_web && tar xzf /mnt/c/Users/corerain/web.tar.gz
cd web && nohup python3 -m http.server 3000 > /dev/null 2>&1 &

# 部署 Vue 前端 (SPA模式)
rm -rf /home/corerain/web_dist && mkdir -p /home/corerain/web_dist/dist
cd /home/corerain/web_dist/dist && tar xzf /mnt/c/Users/corerain/dist.tar.gz
# 用自定义SPA服务器（所有路由返回index.html）
nohup python3 /tmp/vue_server.py > /dev/null 2>&1 &
```

### 5. 设置 Windows 端口转发

```powershell
# 管理员 PowerShell（查 WSL2 IP 后执行）
wsl -d Ubuntu-24.04 hostname -I   # 假设输出 172.17.7.253

netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=9090 connectaddress=172.17.7.253 connectport=9090
netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=3000 connectaddress=172.17.7.253 connectport=3000
netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=5173 connectaddress=172.17.7.253 connectport=5173

# 验证
netsh interface portproxy show all
```

### 6. Windows 防火墙放行

```powershell
netsh advfirewall firewall add rule name="WDOS 9090" dir=in action=allow protocol=tcp localport=9090
netsh advfirewall firewall add rule name="WDOS 3000" dir=in action=allow protocol=tcp localport=3000
netsh advfirewall firewall add rule name="WDOS 5173" dir=in action=allow protocol=tcp localport=5173
```

---

## 三、日常更新部署

```bash
# Mac 上一键部署脚本 deploy.sh
GOOS=linux GOARCH=amd64 go build -o wdos-server-linux ./cmd/api/
gzip -f wdos-server-linux
scp wdos-server-linux.gz chaoyang:C:/Users/corerain/

cd miniapp && flutter build web --no-tree-shake-icons
cd build && rm -f web.tar.gz && tar czf web.tar.gz web/
scp web.tar.gz chaoyang:C:/Users/corerain/

cd ../web && npm run build
cd dist && tar czf ../dist.tar.gz .
scp ../dist.tar.gz chaoyang:C:/Users/corerain/

# 在昭阳执行
ssh chaoyang 'cmd /c "wsl -d Ubuntu-24.04 bash /mnt/c/Users/corerain/deploy.sh"'
```

---

## 四、常用运维命令

```bash
# 查看服务状态
ps aux | grep wdos-server
docker ps

# 查看日志
tail -f /home/corerain/WDOS/wdos.log

# 重启后端
sudo pkill -9 -f wdos-server
cd /home/corerain/WDOS && nohup ./wdos-server-linux > wdos.log 2>&1 &

# 重启 Flutter
sudo pkill -9 -f "http.server 3000"
cd /tmp/wdos_web/web && nohup python3 -m http.server 3000 > /dev/null 2>&1 &

# 查看 MySQL
docker exec -it wdos-mysql mysql -uroot -proot123 wdos_db

# 清空工单数据
docker exec wdos-mysql mysql -uroot -proot123 wdos_db -e "DELETE FROM work_order; DELETE FROM work_order_log; DELETE FROM crip_alarm_raw;"
redis-cli FLUSHDB
```

---

## 五、WSL2 重启后

WSL2 每次重启 IP 会变，需要更新端口转发：

```powershell
# 删旧规则 + 查新 IP + 重新转发
netsh interface portproxy reset
wsl -d Ubuntu-24.04 hostname -I   # 记下新 IP
netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=9090 connectaddress=<新IP> connectport=9090
# 重复 3000, 5173
```

**或者**：添加 `.wslconfig`（`%USERPROFILE%\.wslconfig`）启用镜像网络模式，避免 IP 变化：

```ini
[wsl2]
networkingMode=mirrored
```

---

## 六、已知问题

| 问题 | 处理 |
|------|------|
| WSL2 24h 后被 systemd 杀进程 | 设 `vmIdleTimeout=-1` |
| Windows 更新重启 | 设置活跃时间窗口 |
| Docker 服务起不来 | `sudo service docker start` |
| 端口转发失效 | WSL2 IP 变了，重设 netsh |
