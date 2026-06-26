# WDOS 待修任务行动指南

> 写给接手开发者，三个任务的详细修改方案

---

## 项目信息速览

| 项目 | 路径 |
|------|------|
| 根目录 | `/Users/gmac/Documents/gcode/WDOS` |
| Go 后端入口 | `cmd/api/main.go` |
| alarm 服务 | `internal/service/alarm/service.go` |
| workorder 服务 | `internal/service/workorder/service.go` |
| sla 引擎 | `internal/service/sla/engine.go` |
| WorkOrder 模型 | `internal/model/work_order.go` |
| Flutter APP | `miniapp/lib/` |
| Vue 管理台 | `web/src/views/workorders/WorkOrderList.vue` |
| ECS 服务器 | `106.53.171.89`，SSH: `ssh wdos` |
| ECS 部署目录 | `/opt/wdos/` |
| Docker 容器 | `wdos-mysql` / `wdos-redis` / `wdos-minio` |

### 编译部署命令

```bash
# 编译 Linux 版后端
cd /Users/gmac/Documents/gcode/WDOS
GOOS=linux GOARCH=amd64 go build -o wdos-server-linux ./cmd/api/

# 上传 + 重启（systemd 管理，无需手动 nohup）
gzip -f wdos-server-linux
scp wdos-server-linux.gz wdos:/tmp/
ssh wdos "cd /opt/wdos && sudo gunzip -f /tmp/wdos-server-linux.gz && sudo cp /tmp/wdos-server-linux . && sudo chmod +x wdos-server-linux && sudo systemctl restart wdos-server"

# 编译 Flutter（国内用镜像）
cd miniapp && PUB_HOSTED_URL=https://pub.flutter-io.cn FLUTTER_STORAGE_BASE_URL=https://storage.flutter-io.cn flutter build web --no-tree-shake-icons

# 上传 Flutter
cd build && tar czf web.tar.gz web/ && scp web.tar.gz wdos:/tmp/
ssh wdos "sudo pkill -f 'http.server 3000'; cd /opt/wdos/web && sudo rm -rf web && sudo tar xzf /tmp/web.tar.gz && cd web && sudo nohup python3 -m http.server 3000 > /dev/null 2>&1 &"

# 编译 Vue
cd web && npm run build
cd dist && tar czf /tmp/vue_dist.tar.gz . && scp /tmp/vue_dist.tar.gz wdos:/tmp/
ssh wdos "cd /opt/wdos/vue && sudo rm -rf * && sudo tar xzf /tmp/vue_dist.tar.gz"
```

---

## 任务 1：接单用时 + 处理用时

### 目标
API 返回工单 JSON 时，附带两个计算字段，前端直接显示即可。

### 修改文件
`internal/model/work_order.go`

### 做法
在 WorkOrder 结构体里加两个方法（不是数据库字段，是 JSON 动态计算）：

```go
// AcceptDuration 接单用时（秒）
func (w *WorkOrder) AcceptDuration() int64 {
    if w.AcceptedAt == nil {
        return 0
    }
    return int64(w.AcceptedAt.Time().Sub(w.CreatedAt.Time()).Seconds())
}

// ProcessDuration 处理用时（秒）
func (w *WorkOrder) ProcessDuration() int64 {
    if w.CompletedAt == nil || w.AcceptedAt == nil {
        return 0
    }
    return int64(w.CompletedAt.Time().Sub(w.AcceptedAt.Time()).Seconds())
}
```

然后在 JSON tag 上暴露：

```go
AcceptDuration  int64 `gorm:"-" json:"accept_duration"`
ProcessDuration int64 `gorm:"-" json:"process_duration"`
```

`gorm:"-"` 意思是不要存数据库，纯计算字段。

### 前端显示
Flutter（`order_detail.dart`）和 Vue（`WorkOrderList.vue`）在工单详情里，已经有 `calcDuration` 函数的实现，直接读 `accept_duration` 和 `process_duration` 显示即可。

---

## 任务 2：SLA 超时上报可视化

### 背景
SLA 引擎已经在跑（每 30 秒扫描一次），超时的工单会自动标记 `escalated_level > 0`。现在需要让前端能看到哪些工单已超时。

### 2.1 Vue 工单列表超时标红

**修改文件**：`web/src/views/workorders/WorkOrderList.vue`

状态列改成：如果 `escalated_level > 0`，用红色 tag，文字显示"已超时 L{n}"。

Vue 模板中现在的状态列：
```html
<el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
```

改成：
```html
<el-tag v-if="row.escalated_level > 0" type="danger" size="small">{{ '超时L'+row.escalated_level }}</el-tag>
<el-tag v-else :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
```

### 2.2 SLA 超时通知已在工作
现在 WebSocket 的 `Escalation` 方法已经推送到 `BroadcastToRole`。L1 推 supervisor，L2 推 manager，L3 推 director+admin。Flutter 消息中心的 WebSocket 已连接，后端推了就能收到。

如果 Flutter 端没显示，检查 `main.dart` 里 WebSocket 是否连接了 `/ws/notifications`。

---

## 任务 3：下级数据查看

### 目标
领班/经理/总监能在"我的"页面通过统计卡片下钻，看到部门内每个下级的数据。

### 排查步骤

**第一步：验证 API 返回**

```bash
TOKEN=$(curl -s http://106.53.171.89:9090/api/v1/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"王班长","password":"Test@123"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

curl -s "http://106.53.171.89:9090/api/v1/users/subordinates" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

如果返回空数组 `[]`，问题在后端。如果返回了数据但前端不显示，问题在 Flutter。

**第二步：修后端（如果 API 返回空）**

后端 handler 在 `cmd/api/main.go`，搜 `subordinates`。当前实现可能是简化版。正确的逻辑：

```go
// 查同部门的所有下级
var users []model.User
db.Where("department_id = ? AND role = 'handler'", deptID).Find(&users)
// supervisor/manager 也看同部门的
```

对于 admin/director，返回所有非 admin 用户。

**第三步：修前端（如果 API 有数据但不显示）**

文件 `miniapp/lib/pages/profile.dart` 的 `_loadSubordinates()` 方法已经调用 `/users/subordinates`。检查它是否在正确的时机调用（`_loadUser` 之后）。

---

## 调试技巧

```bash
# 看后端日志
ssh wdos "sudo journalctl -u wdos-server --no-pager -n 50"

# 看进程
ssh wdos "ps aux | grep wdos"

# 看端口
ssh wdos "ss -tlnp | grep -E '9090|3000|5173'"

# 进数据库
ssh wdos "sudo docker exec -it wdos-mysql mysql -uroot -pWDOS186! wdos_db"

# 测试 API（Mac 上跑）
curl -s http://106.53.171.89:9090/health
```
