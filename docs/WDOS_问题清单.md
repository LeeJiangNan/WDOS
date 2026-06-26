# WDOS 问题清单

> 更新：2026-06-25 | 当前阶段：ECS 生产部署

---

## 一、ECS 服务器部署

### 当前状态
| 服务 | 地址 | 状态 |
|------|------|------|
| API 后端 | http://106.53.171.89:9090 | ✅ 正常 |
| Flutter APP | http://106.53.171.89:3000 | ❌ 登录不了 |
| Vue 管理台 | http://106.53.171.89:5173 | ✅ 正常（先改用这个） |
| MySQL | Docker 容器内 | ✅ 正常 |
| Redis | Docker 容器内 | ✅ 正常 |
| MinIO | Docker 容器内 | ✅ 正常 |

### 待解决问题

#### 1. Flutter APP 登录报 ERR_EMPTY_RESPONSE
- **现象**：浏览器访问 `http://106.53.171.89:3000`，输入 admin/Admin@123，控制台报 `POST http://106.53.171.89:9090/api/v1/auth/login net::ERR_EMPTY_RESPONSE`
- **排查结果**：
  - Mac 上用 curl 直接调 API 正常返回 token ✅
  - CORS 预检 OPTIONS 返回 204 ✅
  - 后端日志里完全没有收到登录请求（请求没到达服务器）
  - 后端已改为 debug 模式，CORS 全放行
- **可能原因**：
  - Flutter 编译时 pub.dev 被墙，无法 clean rebuild
  - 浏览器缓存了旧的 Service Worker
  - Flutter Web HTML 渲染器的 XMLHttpRequest 跨域问题
- **解决思路**：配置 Flutter 国内镜像重新编译，或用 `--web-renderer canvaskit` 试试

#### 2. 图片存储
- 当前：MinIO（本地容器，未挂载 volume，重启丢数据）
- 计划：迁移到腾讯云 COS（代码已改好，main.go 回退后暂未启用）

#### 3. 进程守护
- 当前：nohup 手动启动，进程挂了不会自动重启
- 需要：写 systemd service 文件

---

## 二、本地开发环境问题

### 已解决
| 问题 | 解决方式 |
|------|---------|
| Docker Hub 被墙拉不到镜像 | 配腾讯云镜像加速 `mirror.ccs.tencentyun.com` |
| Flutter pub.dev 被墙 | 设 `PUB_HOSTED_URL` 和 `FLUTTER_STORAGE_BASE_URL` 国内镜像 |
| Mac 编译 Linux 二进制 | `GOOS=linux GOARCH=amd64 go build` |
| ECS SSH 连接 | 用户名是 ubuntu，密钥在 `/Users/gmac/Documents/gcode/yunfuwu/WDOS.pem` |

---

## 三、数据库/数据问题

### 已解决
| 问题 | 解决方式 |
|------|---------|
| MySQL 中文名乱码 | 容器启动加 `--character-set-server=utf8mb4` |
| 用户密码不匹配 | 导入后 UPDATE 重置为明文 |
| 工单数为0 | 工单日期是历史，UPDATE created_at=NOW() |

### 待注意
- 昭阳数据库 dump 在 Mac 本地 `/tmp/wdos_dump.sql`
- ECS 当前是 CRIP 直推的数据（53条），不是历史数据

---

## 四、代码版本问题

- **当前 git 状态**：`72ebad7 backup: 修bug前的最后状态`
- **问题**：ECS 上的 `main.go` 是从 git 恢复的旧版本，缺少以下改动：
  - resolveViewer（view_as 身份切换）
  - 多部门支持（department_ids）
  - COS 图片上传
  - 统计时间范围（过去24h/7天/30天）
  - ListByStatus/ListAll 的 deptIDs 参数
  - 排班按部门过滤
  - CORS 模式区分
  - registerRoutes 的 rdb/minioClient 参数

- **已更新的文件（不受 git 回退影响）**：
  - `internal/service/alarm/service.go` — MinIO→COS 上传
  - `internal/service/stats/service.go` — 统计逻辑
  - `internal/service/workorder/service.go` — 一线只看自己工单
  - `internal/service/schedule/service.go` — 排班事务+部门过滤
  - `internal/service/auth/service.go` — 明文密码
  - `internal/pkg/jwt/jwt.go` — 多部门 JWT
  - `internal/pkg/config/config.go` — COS 配置
  - `internal/model/user_department.go` — 用户多部门关联表
  - Flutter/Vue 前端文件

---

## 五、架构/网络问题

### ECS 端口开放情况（腾讯云防火墙）
| 端口 | 状态 | 用途 |
|------|------|------|
| 22 | ✅ | SSH |
| 9090 | ✅ | API |
| 3000 | ✅ | Flutter |
| 5173 | ✅ | Vue |

### 后续要停的服务
- 腾讯云 SCF：`wdos-callback-buffer`（已不需要，CRIP 直推 ECS）
- 腾讯云 COS：`wdos-callback-1446142760`（Poller 中转用的，ECS 直推后不需要）
- 昭阳笔记本上的所有服务

---

## 六、常用命令

```bash
# SSH 连接 ECS
ssh wdos
# 或
ssh -i /Users/gmac/Documents/gcode/yunfuwu/WDOS.pem ubuntu@106.53.171.89

# 查看服务状态
curl http://106.53.171.89:9090/health    # 后端
curl http://106.53.171.89:3000/          # Flutter
curl http://106.53.171.89:5173/login     # Vue

# 重启后端
ssh wdos "sudo pkill -f wdos-server; sleep 1; cd /opt/wdos && sudo nohup ./wdos-server-linux > wdos.log 2>&1 &"

# 查看数据库
ssh wdos "sudo docker exec wdos-mysql mysql -uroot -pWDOS186! wdos_db"
```
