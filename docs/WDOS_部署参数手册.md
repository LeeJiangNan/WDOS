# WDOS 部署参数手册

> 最后更新：2026-06-25

---

## 一、ECS 服务器

| 项目 | 值 |
|------|-----|
| 实例名称 | WDOS-1 |
| 实例 ID | lhins-mx8mdjb5 |
| 公网 IPv4 | **106.53.171.89** |
| 地域 | 广州六区 |
| 系统 | Ubuntu 24.04 LTS |
| 配置 | 2核 4GB 80GB SSD 8Mbps |
| 用户名 | `ubuntu` |
| 密码 | `WDOS186!` |
| SSH 密钥 | `lhkp-ctdtxjtv` |
| 密钥文件 | `/Users/gmac/Documents/gcode/yunfuwu/WDOS.pem` |

### SSH 连接方式

```bash
# Mac 本地已配置 SSH alias
ssh wdos

# 等价于
ssh -i /Users/gmac/Documents/gcode/yunfuwu/WDOS.pem ubuntu@106.53.171.89
```

`~/.ssh/config` 中的配置：
```
Host wdos
  Hostname 106.53.171.89
  User ubuntu
  IdentityFile /Users/gmac/Documents/gcode/yunfuwu/WDOS.pem
```

---

## 二、Docker 容器

| 容器名 | 镜像 | 端口 | 账号/密码 |
|--------|------|------|-----------|
| wdos-mysql | mysql:8.0 | 3306 | root / WDOS186! |
| wdos-redis | redis:7-alpine | 6379 | 无密码 |
| wdos-minio | minio/minio | 9000/9001 | minioadmin / minioadmin |

### docker-compose 位置
```
/opt/wdos/docker-compose.yml
```

### 管理命令
```bash
ssh wdos "sudo docker ps"                        # 查看容器
ssh wdos "sudo docker compose -f /opt/wdos/docker-compose.yml restart"  # 重启
ssh wdos "sudo docker exec -it wdos-mysql mysql -uroot -pWDOS186! wdos_db"  # 进MySQL
```

---

## 三、应用部署路径

| 组件 | 服务器路径 | 启动方式 |
|------|-----------|---------|
| 后端二进制 | `/opt/wdos/wdos-server-linux` | `nohup ./wdos-server-linux > wdos.log 2>&1 &` |
| 后端日志 | `/opt/wdos/wdos.log` | - |
| 后端配置 | `/opt/wdos/config/config.yaml` | - |
| Flutter Web | `/opt/wdos/web/web/` | `python3 -m http.server 3000` |
| Vue Web | `/opt/wdos/vue/` | `python3 /opt/wdos/vue_server.py` (SPA模式，端口5173) |

### 更新部署流程

```bash
# 1. 本地编译
cd /Users/gmac/Documents/gcode/WDOS
GOOS=linux GOARCH=amd64 go build -o wdos-server-linux ./cmd/api/
gzip -f wdos-server-linux

# 2. 上传
scp wdos-server-linux.gz wdos:/tmp/
# Flutter
cd miniapp/build && tar czf web.tar.gz web/ && scp web.tar.gz wdos:/tmp/
# Vue
cd web/dist && tar czf /tmp/vue_dist.tar.gz . && scp /tmp/vue_dist.tar.gz wdos:/tmp/

# 3. 在服务器上替换部署
ssh wdos "sudo bash -s" << 'EOF'
# 后端
cp /tmp/wdos-server-linux.gz /opt/wdos/ && cd /opt/wdos
gunzip -f wdos-server-linux.gz && chmod +x wdos-server-linux
pkill -f wdos-server && sleep 1
nohup ./wdos-server-linux > wdos.log 2>&1 &

# Flutter
cd /opt/wdos/web && rm -rf web && tar xzf /tmp/web.tar.gz
pkill -f "http.server 3000" && sleep 1
cd web && nohup python3 -m http.server 3000 > /dev/null 2>&1 &

# Vue
cd /opt/wdos/vue && rm -rf * && tar xzf /tmp/vue_dist.tar.gz
pkill -f vue_server && sleep 1
nohup python3 /opt/wdos/vue_server.py > /dev/null 2>&1 &
EOF
```

---

## 四、腾讯云相关

| 项目 | 值 |
|------|-----|
| COS Bucket | wdos-callback-1446142760 |
| COS 地域 | ap-guangzhou |
| COS SecretId | ${COS_SECRET_ID} |
| COS SecretKey | ${COS_SECRET_KEY} |
| SCF 函数名 | wdos-callback-buffer（待停用） |

---

## 五、CRIP 回调配置

| 项目 | 值 |
|------|-----|
| 当前回调地址 | http://106.53.171.89:9090/api/v1/callback/crip |
| 方法 | POST |
| Content-Type | application/json |
| 签名校验 | 无（已关闭） |

---

## 六、登录账号（测试用）

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | Admin@123 | 管理员 |

---

## 七、Docker 镜像加速

ECS 上 `/etc/docker/daemon.json`：
```json
{
  "registry-mirrors": [
    "https://mirror.ccs.tencentyun.com",
    "https://docker.m.daocloud.io"
  ]
}
```

---

## 八、当前 ECS config.yaml

```yaml
server:
  port: "9090"
  mode: debug          # 注意：改成了debug，CORS全放行

database:
  host: "127.0.0.1"
  port: "3306"
  user: "root"
  password: "WDOS186!"
  name: "wdos_db"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
  prefix: "wdos:"

minio:
  endpoint: "127.0.0.1:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  bucket: "wdos"
  use_ssl: false

jwt:
  secret: "wdos-prod-secret-2026"
  expire_seconds: 86400

sla:
  accept_l1_seconds: 30
  accept_l2_seconds: 150
  accept_l3_seconds: 300
  process_l1_seconds: 60
  process_l2_seconds: 300
  process_l3_seconds: 600
```

---

## 九、本地开发环境

| 项目 | 路径 |
|------|------|
| 项目根目录 | `/Users/gmac/Documents/gcode/WDOS` |
| Go 后端 | `cmd/api/main.go` |
| Flutter APP | `miniapp/` |
| Vue 管理台 | `web/` |
| 文档 | `docs/` |
| SCF/Poller | `scf/`, `tools/poller/` |
| SSH 密钥 | `/Users/gmac/Documents/gcode/yunfuwu/WDOS.pem` |

### 本地 Docker（跟 ECS 一样的容器）
```
MySQL: localhost:3307 (root/root123)
Redis: localhost:6380
MinIO: localhost:9000 (minioadmin/minioadmin)
```
