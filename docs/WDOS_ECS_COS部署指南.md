# WDOS ECS + COS 部署指南

> **环境**：腾讯云 ECS + COS + 公网 IP
> **用途**：生产环境正式使用
> **更新**：2026-06-24

---

## 一、架构概览

```
┌─────────────────────────────────────────────────────┐
│                    腾讯云                             │
│                                                     │
│  CRIP ──▶ ECS:9090/api/v1/callback/crip             │
│              │                                      │
│       ┌──────┼──────┐                               │
│       ▼      ▼      ▼                               │
│     Go后端  Redis  MySQL(Docker)                    │
│       │                                             │
│       ├── 图片处理 → COS (公网URL)                   │
│       ├── Flutter APP :3000                         │
│       └── Vue Web    :5173                          │
│                                                     │
│  COS ─── 存图片/附件 ─── 公网 URL ─── 前端直读       │
│       └── 生命周期: 30天→低频, 180天→删除            │
└─────────────────────────────────────────────────────┘
```

**与昭阳版的区别**：

| 组件 | 昭阳版 | ECS版 |
|------|--------|-------|
| MinIO | ✅ 需要 | ❌ 砍掉，用 COS |
| SCF | ✅ 中转回调 | ❌ 砍掉，直推 ECS |
| COS (回调缓冲) | ✅ Poller 轮询 | ❌ 砍掉，直推 ECS |
| COS (图片存储) | ❌ | ✅ 替换 MinIO |
| 端口转发 | 手动 netsh | 不需要 |
| 网络 | 家庭宽带 NAT | 公网固定 IP |

---

## 二、买什么配置

### ECS（腾讯云轻量应用服务器）

| 项目 | 建议配置 |
|------|---------|
| CPU | 4核 |
| 内存 | 8GB |
| 系统盘 | 80GB SSD |
| 带宽 | 8Mbps |
| 系统 | Ubuntu 22.04 或 24.04 |
| 地域 | 广州（跟 SCF/COS 同地域） |
| 月费 | ~¥130 |

### COS（对象存储）

使用现有的 COS bucket（`wdos-callback-buffer-xxx`），或在同地域新建一个 `wdos-images`。

| 项目 | 设置 |
|------|------|
| 存储类型 | 标准存储 |
| 访问权限 | 公有读（图片需要前端直读） |
| 生命周期 | 30天→低频, 180天→删除 |
| 防盗链 | 可选，限制 Referer |
| 费用 | 50GB 以内免费 |

### 域名（可选）

```
wdos.kunyun.com → ECS 公网 IP
```

---

## 三、COS 配置

### 1. Bucket 设置

```
控制台 → 对象存储 → 创建 Bucket
  名称: wdos-images
  地域: 广州
  访问权限: 公有读私有写
```

### 2. 生命周期规则

```
规则名称: auto-clean
应用范围: 整个 Bucket
规则:
  ├── 30天后 → 转为低频存储
  └── 180天后 → 删除
```

### 3. 防盗链（可选，推荐）

```
Referer 白名单:
  - wdos.kunyun.com
  - 100.107.124.26 (Tailscale)
  - ECS 公网 IP
```

---

## 四、ECS 初始化

### 1. SSH 登录 + 基础配置

```bash
ssh root@<ECS公网IP>

# 换国内镜像源（加速 apt/docker）
curl -fsSL https://mirrors.tencentyun.com/install.sh | bash

# 安装基础包
apt update && apt install -y curl wget git vim

# 设置时区
timedatectl set-timezone Asia/Shanghai
```

### 2. 安装 Docker

```bash
curl -fsSL https://get.docker.com | bash
systemctl enable docker
systemctl start docker
```

### 3. 创建项目目录

```bash
mkdir -p /opt/wdos
cd /opt/wdos
```

### 4. 配置 docker-compose.yml

```yaml
version: '3'
services:
  mysql:
    image: mysql:8.0
    container_name: wdos-mysql
    environment:
      MYSQL_ROOT_PASSWORD: <强密码>
      MYSQL_DATABASE: wdos_db
    ports:
      - "3306:3306"
    volumes:
      - ./mysql_data:/var/lib/mysql
    restart: always
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci

  redis:
    image: redis:7-alpine
    container_name: wdos-redis
    ports:
      - "6379:6379"
    restart: always
```

```bash
docker compose up -d
```

### 5. 配置 config.yaml

```yaml
# /opt/wdos/config/config.yaml
server:
  mode: release   # 生产模式（CORS 白名单）
  port: "9090"

database:
  host: 127.0.0.1
  port: "3306"
  user: root
  password: <强密码>
  dbname: wdos_db

redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
  prefix: "wdos:"

cos:
  secret_id: "<腾讯云 SecretId>"
  secret_key: "<腾讯云 SecretKey>"
  bucket: "wdos-images-<APPID>"
  region: "ap-guangzhou"

jwt:
  secret: "<随机生成64位字符串>"
  expire_seconds: 86400

sla:
  accept_l1_seconds: 30
  accept_l2_seconds: 150
  accept_l3_seconds: 300
  process_l1_seconds: 60
  process_l2_seconds: 300
  process_l3_seconds: 600

crip:
  openapi_base: "https://crip-api.example.com"
  openapi_app_id: "<CRIP AppID>"
  openapi_app_secret: "<CRIP AppSecret>"
  callback_secret: "<回调签名密钥>"

wechat:
  app_id: "<微信小程序AppID>"
  app_secret: "<微信小程序AppSecret>"
```

### 6. 修改代码：图片存储改用 COS

需要改 `alarm/service.go`，图片上传从 MinIO 切到 COS：

```go
// 替换 MinIO PutObject
// s.minio.PutObject(...) → COS SDK PutObject(...)

// 上传后返回 COS 公网 URL
alarmPicLocal = fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s",
    bucket, region, objectName)
```

### 7. 部署应用

```bash
# 从 Mac 传到 ECS
cd /path/to/WDOS
GOOS=linux GOARCH=amd64 go build -o wdos-server ./cmd/api/
scp wdos-server root@<ECS_IP>:/opt/wdos/

# Flutter
cd miniapp && flutter build web --no-tree-shake-icons
cd build && tar czf web.tar.gz web/
scp web.tar.gz root@<ECS_IP>:/opt/wdos/

# Vue
cd ../web && npm run build
cd dist && tar czf ../dist.tar.gz .
scp ../dist.tar.gz root@<ECS_IP>:/opt/wdos/
```

```bash
# 在 ECS 上
cd /opt/wdos

# 后端
chmod +x wdos-server
nohup ./wdos-server > wdos.log 2>&1 &

# Flutter (用 nginx 或 systemd)
mkdir -p /opt/wdos/flutter && cd /opt/wdos/flutter
tar xzf ../web.tar.gz
# 用 systemd service 保持运行

# Vue (SPA 模式)
mkdir -p /opt/wdos/vue && cd /opt/wdos/vue
tar xzf ../dist.tar.gz
```

### 8. 防火墙开放端口

```bash
# 腾讯云控制台 → 安全组 → 添加入站规则
TCP 9090  (API)
TCP 3000  (Flutter)
TCP 5173  (Vue)
TCP 80    (如果用了域名)
TCP 443   (如果用了 HTTPS)
```

---

## 五、Systemd 服务（保持进程存活）

```ini
# /etc/systemd/system/wdos-server.service
[Unit]
Description=WDOS API Server
After=network.target docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/wdos
ExecStart=/opt/wdos/wdos-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/wdos-flutter.service
[Unit]
Description=WDOS Flutter Web
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/wdos/flutter/web
ExecStart=/usr/bin/python3 -m http.server 3000
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
systemctl enable wdos-server wdos-flutter
systemctl start wdos-server wdos-flutter
```

---

## 六、数据迁移（从昭阳到 ECS）

```bash
# 1. 在昭阳导出数据库
docker exec wdos-mysql mysqldump -uroot -proot123 wdos_db > /tmp/wdos_backup.sql
scp chaoyang:C:/Users/corerain/wdos_backup.sql .

# 2. 导入到 ECS
scp wdos_backup.sql root@<ECS_IP>:/opt/wdos/
docker exec -i wdos-mysql mysql -uroot -p<新密码> wdos_db < /opt/wdos/wdos_backup.sql

# 3. 迁移 COS/ MinIO 图片（如果有历史图片）
# 用 COS 控制台上传，或用 coscli 批量迁移
```

---

## 七、日常运维

### 更新部署

```bash
# 本地编译 → 上传 → 重启
cd /path/to/WDOS
GOOS=linux GOARCH=amd64 go build -o wdos-server ./cmd/api/
scp wdos-server root@<ECS_IP>:/opt/wdos/
ssh root@<ECS_IP> "systemctl restart wdos-server"

# Flutter/Vue 同理
```

### 数据库备份

```bash
# 每天凌晨3点自动备份
# crontab -e
0 3 * * * docker exec wdos-mysql mysqldump -uroot -p<密码> wdos_db | gzip > /opt/wdos/backups/wdos_$(date +\%Y\%m\%d).sql.gz
```

### 日志查看

```bash
journalctl -u wdos-server -f   # systemd 日志
tail -f /opt/wdos/wdos.log     # 应用日志
docker logs wdos-mysql         # MySQL 日志
```

### 监控

```bash
# CPU/内存/磁盘
htop
df -h
docker stats
```

---

## 八、费用预估

| 项目 | 月费 |
|------|------|
| ECS 轻量 4核8G 8M | ¥130 |
| COS 存储 + 流量 | ¥0-5（大概率免费额度内） |
| 域名（可选） | ¥5-10 |
| **合计** | **~¥140/月** |
