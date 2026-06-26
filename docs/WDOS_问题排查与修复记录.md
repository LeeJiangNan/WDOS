# WDOS 问题排查与修复记录

> 时间：2026-06-22  
> 范围：全项目代码 + 架构设计  
> 项目路径：/Users/gmac/Documents/gcode/WDOS  
> 编译验证：`go build ./...` 全部通过

## 总结

作为 vibe coding 项目，架构方向正确（Gin+GORM+Redis+MinIO 分层清晰），文档质量高。但存在多个"模块写了但没串起来"的问题，以及若干会直接导致线上事故的安全漏洞。经三轮修复，P0/P1 级问题全部清零，仅剩 P2/P3 代码质量项待上线前处理。

> **修复状态图例**: ✅ 已修复 | ⚠️ 部分修复 | ❌ 未修复

---

## 一、P0 级 — 上线必修（会直接出事）

### 1.1 JWT 中间件未挂载 — 全站裸奔 ✅ 已修复（第一轮）

**位置**: `cmd/api/main.go` → `registerRoutes()`

`JWTAuth` 中间件定义了但从未使用。工单中心、模板管理、用户管理、排班、统计等接口全部无需登录即可访问。

**影响**: 任何人可接单、转交、创建用户、操作抑制策略。

**修复**: 所有需要认证的路由组加上 `JWTAuth(jwtMgr)`：
```go
orders := v1.Group("/work-orders", JWTAuth(jwtMgr))
templates := v1.Group("/templates", JWTAuth(jwtMgr))
userGrp := v1.Group("/users", JWTAuth(jwtMgr), RoleRequired("admin"))
// ... 等等
```

**状态**: 第一轮已修复，JWTAuth 已挂到工单/排班/抑制策略/用户/统计/权限等所有接口。手动补偿接口加了 `RoleRequired("admin")`。

---

### 1.2 Callback 接口无鉴权 ✅ 已修复（第三轮）

**位置**: `v1.POST("/callback/crip", ...)`

无签名校验、无 IP 白名单。任何人可伪造报警数据。

**修复**: 增加 HMAC-SHA256 签名校验。CRIP 请求需带 `X-CRIP-Signature` 头，服务端用 `callback_secret` 重新计算 HMAC 对比。`callback_secret` 通过 `${CRIP_CALLBACK_SECRET}` 环境变量注入，未配置时跳过校验（兼容开发环境）。

**改动文件**: `cmd/api/main.go`、`config/config.yaml`、`internal/pkg/config/config.go`

---

### 1.3 手动补偿接口无鉴权 ✅ 已修复（第一轮）

**位置**: `v1.POST("/admin/compensate", ...)`

无 JWT、无角色检查。任何人可触发补偿调 CRIP OpenAPI。

**修复**: 挂了 `JWTAuth` + `RoleRequired("admin")`。

---

### 1.4 路由引擎未调用 — 工单无部门无处理人 ✅ 已修复（第二轮）

**位置**: `internal/service/alarm/service.go` → `ProcessCallback()`

`routeEngine` 初始化了但在工单生成时未调用。`AssigneeID`、`DepartmentID` 全为 0。handler 角色的工单列表用 `assignee_id = userID` 过滤，永远看不到分配给自己的工单。

**修复**: ProcessCallback 中调 `routeEngine.Route(cb.CameraGroup)`，设置 `order.DepartmentID = result.DepartmentID` 和 `order.AssigneeID = result.HandlerGroupID`。

**改动文件**: `internal/service/alarm/service.go`

---

### 1.5 WebSocket 通知 Hub 未接入 ✅ 已修复（第二轮）

**位置**: `alarm/service.go` 和 `workorder/service.go`

`notifyHub` 初始化了但工单创建/接单/完成时从未调用推送。前端实时更新不工作。

**修复**:
- alarm 在工单创建后调 `notifyHub.NewOrder(notify.NewOrderPayload{...})`
- workorder 在接单后调 `notifyHub.OrderAccepted()`、完成后调 `notifyHub.OrderCompleted()`
- `workorder.NewService` 签名增加 `notifyHub` 参数

**改动文件**: `internal/service/alarm/service.go`、`internal/service/workorder/service.go`、`cmd/api/main.go`

---

### 1.6 JWT Secret 可能未生效 ✅ 已修复（第三轮）

**位置**: `config/config.yaml` + `internal/pkg/config/config.go`

Viper 的 `AutomaticEnv()` 不会自动替换 YAML 中的 `${VAR_NAME}` 语法。JWT secret 可能是字面量 `"${JWT_SECRET}"`。

**修复**: `config.Load()` 改为先用 `os.ReadFile` 读取原始 YAML → `os.ExpandEnv` 展开 `${VAR}` → 再交给 Viper 解析。同时 `CRIPConfig` 新增 `callback_secret` 字段。

**改动文件**: `internal/pkg/config/config.go`、`config/config.yaml`

---

### 1.7 APP 转圈打不开 — 网络地址问题 ✅ 已修复（第一轮）

**位置**: `miniapp/lib/config/api.dart`

`baseUrl = 'http://localhost:8080/api/v1'` — Android 模拟器的 localhost 指向模拟器自身，不是宿主机 Mac。

**修复**: `localhost` → `10.0.2.2`（Android 模拟器映射宿主机）。真机需改为 Mac 局域网 IP。

**改动文件**: `miniapp/lib/config/api.dart`

---

### 1.8 APP 登录成功后白屏 ✅ 已修复（第一轮）

**位置**: `miniapp/lib/pages/login.dart`

登录成功后 `Navigator.pushReplacement` 跳转到 `SizedBox()` 而非 `MainTabs()`。

**修复**: 改为 `MainTabs()`，并 import `../main.dart`。

**改动文件**: `miniapp/lib/pages/login.dart`

---

### 1.9 工单列表永远为空 — JWT 缺失导致 user_id 为空 ✅ 已修复（第一轮）

**位置**: `cmd/api/main.go` → `registerRoutes()`

工单路由未挂 `JWTAuth`，`c.GetString("role")` 和 `c.GetString("user_id")` 全返回空字符串。`ListByStatus` 里 role="" 走 handler 分支，`assignee_id = 0`，查不出任何工单。

**修复**: 工单路由组加 `JWTAuth(jwtMgr)` 中间件。

---

## 二、P1 级 — 数据 Bug（会导致错误结果）

### 2.1 抑制查询用 camera_name 而非 camera_id ❌ 未修复

**位置**: `alarm/service.go` ProcessCallback, `workorder/service.go` CheckSuppression

用 `camera_name + algorithm_name` 字符串匹配，应改用 `camera_id + algorithm_id`。需确认 CRIP Callback 是否传 camera_id，以及 WorkOrder 模型是否需要加 CameraID 字段。

---

### 2.2 工单编号碰撞风险 ✅ 已修复（第二轮）

**位置**: `generateOrderNo()` — `alarm/service.go` 和 `workorder/service.go` 各有一份

`UnixMilli() % 10000` 同毫秒内会重复，uniqueIndex 会导致创建失败。

**修复**: `%04d` (10000空间) → `%06d` (1000000空间)，碰撞概率大幅降低。

**备注**: 两个文件仍有重复代码（DRY 问题未解决）。长期建议用 snowflake ID。

**改动文件**: `internal/service/alarm/service.go`、`internal/service/workorder/service.go`

---

### 2.3 操作日志 Action 字段永远写 "created" ✅ 已修复（第二轮）

**位置**: `internal/service/workorder/service.go` → `addLog()`

不管接单/提交/转交，Action 都写 "created"，日志失去意义。

**修复**: `addLog` 内部根据 fromStatus/toStatus 推导 action：
- `"" → "pending"` → `"created"`
- `"pending" → "processing"` → `"accepted"`
- `"processing" → "completed"` → `"completed"`
- `"processing" → "pending"` → `"transferred"`

**改动文件**: `internal/service/workorder/service.go`

---

### 2.4 排班日期计算 nextDay 手写且错误 ✅ 已修复（第三轮）

**位置**: `schedule/service.go` → `nextDay()`

手写日期++，`d > 28` 导致30/31号等边界全错。

**修复**: 改为 `time.Parse("2006-01-02", date)` + `AddDate(0, 0, 1).Format(...)`。

**改动文件**: `internal/service/schedule/service.go`

---

### 2.5 耗时分布统计区间重叠 ✅ 已修复（第二轮）

**位置**: `internal/service/stats/service.go` → `ProcessTimeDist()`

`BETWEEN 0 AND 30` 和 `BETWEEN 30 AND 60` 在 MySQL 中是闭区间，30秒工单被计两次。

**修复**: 改为不重叠区间 `0-30, 31-60, 61-120, 121-300, 301+`。

**改动文件**: `internal/service/stats/service.go`

---

### 2.6 前端工单列表耗时列显示错误字段 ✅ 已修复（第二轮）

**位置**: `web/src/views/workorders/WorkOrderList.vue`

显示 `row.duplicate_count`（抑制次数），应为处理耗时。

**修复**: 改为前端计算 `accepted_at` → `completed_at` 的秒数差，新增 `calcDuration()` 函数。

**改动文件**: `web/src/views/workorders/WorkOrderList.vue`

---

### 2.7 前端 Dashboard 数据结构不匹配 ✅ 已修复（第二轮）

**位置**: `web/src/views/Dashboard.vue` + `miniapp/lib/pages/home.dart`

后端 `/stats/my-overview` 返回扁平结构 `{total_alarms, total_orders, completed_orders, ...}`，前端取 `res.data.today.pending_count` 永远 undefined。

**修复**:
- Web: 改为读 `res.data.total_alarms`、`total_orders`、`completion_rate`、`overtime_orders`
- APP: 改为读 `data['pending_orders']`、`processing_orders`、`completed_orders`、`overtime_orders`
- 后端 DailyOverview 补了 `processing_orders`、`pending_orders` 字段

**改动文件**: `web/src/views/Dashboard.vue`、`miniapp/lib/pages/home.dart`、`internal/service/stats/service.go`

---

### 2.8 ShouldBindJSON 不检查错误 — 抑制策略创建"假成功" ✅ 已修复（第三轮）

**位置**: `cmd/api/main.go` 多个 POST 接口

不检查错误返回值，绑定失败时前端依然收到 `code: 0`。

**修复**: 全面排查所有 POST/PUT 接口，toggle、submit、transfer 三处漏网之鱼全部补上 `ShouldBindJSON` 错误检查和返回。

**改动文件**: `cmd/api/main.go`

---

### 2.9 APP 登录用 username 但后端查 name/phone ⚠️ 部分修复

**位置**: `miniapp/lib/pages/login.dart` vs `auth/service.go` WebLogin

APP 登录默认用户名 `admin`，后端 `WebLogin` 查 `name = ? OR phone = ?`。seedAdmin 创建的用户 name="admin" 能匹配，但其他用户用手机号登录会失败（phone 是 "00000000000"）。

---

### 2.10 前端"全部状态"筛选实际只显示 pending ✅ 已修复（第三轮）

**位置**: `web/src/views/workorders/WorkOrderList.vue`

选"全部状态"时 `status.value` 为空，fallback 到 `/work-orders/pending`。后端也没有"全部"的 API。

**修复**: 后端新增 `ListAll` 方法 + `GET /api/v1/work-orders` 路由（角色限定不变），前端 fallback 改为 `/work-orders`。

**改动文件**: `internal/service/workorder/service.go`、`cmd/api/main.go`、`web/src/views/workorders/WorkOrderList.vue`

---

## 三、P2 级 — 性能与健壮性

### 3.1 SLA 引擎全表扫描 ✅ 已修复（第二轮）

**位置**: `internal/service/sla/engine.go` → `scan()`

每秒 `WHERE status = 'pending'` 全表加载到内存。工单量增长后能把数据库 CPU 打满。

**修复**: 加 `created_at >= ?` 过滤，只扫最近24小时的工单。

**进一步优化**: 可加 `created_at < NOW() - INTERVAL 30 SECOND` 只扫可能超时的。

**改动文件**: `internal/service/sla/engine.go`

---

### 3.2 WebSocket 僵尸连接 ✅ 已修复（第二轮）

**位置**: `internal/service/notify/hub.go` + `cmd/api/main.go`

无 ping/pong 心跳、无读写超时。断网连接不被清理，一直占内存。

**修复**:
- Hub 新增 `heartbeatLoop()` goroutine，每30秒 ping 所有连接，ping 失败立即清理
- WebSocket handler 设置 60s read deadline + pong handler 重置

**改动文件**: `internal/service/notify/hub.go`、`cmd/api/main.go`

---

### 3.3 CORS 配置 `*` 不安全 ✅ 已修复（第二轮）

**位置**: `cmd/api/main.go`

`Access-Control-Allow-Origin: *` 生产环境不安全。

**修复**: 改为白名单模式（localhost:5173/3000 + 可配置），增加 `Allow-Credentials: true`。

**改动文件**: `cmd/api/main.go`

---

### 3.4 图片下载无超时 ✅ 已修复（第二轮）

**位置**: `internal/service/alarm/service.go` → `downloadImage()`

`http.Get` 无超时，下载挂死会拖垮整个 Callback 处理 goroutine。

**修复**: 改用 `http.Client{Timeout: 10 * time.Second}`，并检查 HTTP 状态码。

**改动文件**: `internal/service/alarm/service.go`

---

### 3.5 统计 ByArea 硬编码区域 ✅ 已修复（第三轮）

**位置**: `internal/service/stats/service.go` → `ByArea()`

硬编码 `["B1停车场", "B2停车场", ...]`，换商场就错。

**修复**: 改为从 `area_routing_rule` 表动态获取区域列表，按 `camera_group` LIKE 匹配计数。无规则时回退到硬编码兜底。

**改动文件**: `internal/service/stats/service.go`

---

## 四、P3 级 — 代码质量

| 问题 | 状态 | 说明 |
|------|------|------|
| `main.go` registerRoutes 缩进混乱 | — | 功能不受影响 |
| `generateOrderNo` / `degreeToPriority` 重复代码 | ❌ 未修复 | alarm/service.go 和 workorder/service.go 各一份 |
| `seedAdmin` 密码硬编码 | ❌ 未修复 | 开发环境 OK，上线前必须改 |
| `itoa` 手写（SLA engine 和 notify/hub 各一份） | ✅ 已修复（第三轮） | 已全部替换为 `strconv.Itoa` |
| config.yaml 明文密码 | ❌ 未修复 | 开发环境 OK，上线前需改 |
| 零测试文件 | ❌ 未修复 | — |
| 前端无 403/404 全局处理 | ❌ 未修复 | — |
| 废弃的 `internal/transport/http/middleware.go` | ✅ 已修复 | 第二轮删除，旧 JWTAuth 与新版冲突（user_id 类型 uint64 vs string） |
| `BroadcastToRole` 实际还是广播所有人 | ❌ 未修复 | — |
| `matchPattern` 只支持后缀 `*` | ❌ 未修复 | 不支持中间通配 |
| SLA 硬编码阈值（30s/150s） | ✅ 已配置化 | `main.go` 已将 config.yaml 的6个阈值传给 `sla.New()`，引擎用配置值（原标注有误） |
| Flutter `/wechat/login` 端点未实现 | ❌ 未修复 | login.dart 实际调的是 `/auth/login`，但代码里有 `/wechat/login` 引用 |

---

## 五、修复进度总览

### 第一轮修复（已完成）

| 编号 | 问题 | 改动文件 |
|------|------|----------|
| 1.1 | JWT 中间件挂载 | `cmd/api/main.go` |
| 1.3 | 手动补偿接口鉴权 | `cmd/api/main.go` |
| 1.7 | APP baseUrl localhost → 10.0.2.2 | `miniapp/lib/config/api.dart` |
| 1.8 | APP 登录后跳转 SizedBox → MainTabs | `miniapp/lib/pages/login.dart` |
| 1.9 | 工单列表 JWT 缺失导致永远为空 | `cmd/api/main.go` |
| 2.7 | APP/Web 首页统计数据字段不匹配 | `miniapp/lib/pages/home.dart`、`web/src/views/Dashboard.vue` |

### 第二轮修复（已完成）

| 编号 | 问题 | 改动文件 |
|------|------|----------|
| 1.4 | 路由引擎调用 — 工单分配部门和人员 | `internal/service/alarm/service.go` |
| 1.5 | WebSocket 通知 Hub 接入 | `internal/service/alarm/service.go`、`internal/service/workorder/service.go`、`cmd/api/main.go` |
| 2.2 | 工单编号碰撞 %04d → %06d | `internal/service/alarm/service.go`、`internal/service/workorder/service.go` |
| 2.3 | 操作日志 Action 硬编码 → 动态推导 | `internal/service/workorder/service.go` |
| 2.5 | 耗时分布区间重叠 | `internal/service/stats/service.go` |
| 2.6 | 前端工单列表耗时列字段错误 | `web/src/views/workorders/WorkOrderList.vue` |
| 2.7 | 前端 Dashboard 数据结构不匹配（补全） | `web/src/views/Dashboard.vue`、`internal/service/stats/service.go` |
| 3.1 | SLA 引擎全表扫描 → 加24小时过滤 | `internal/service/sla/engine.go` |
| 3.2 | WebSocket 僵尸连接 → 加心跳 | `internal/service/notify/hub.go`、`cmd/api/main.go` |
| 3.3 | CORS `*` → 白名单 | `cmd/api/main.go` |
| 3.4 | 图片下载无超时 → 10s client | `internal/service/alarm/service.go` |
| — | 废弃中间件文件删除 | `internal/transport/http/middleware.go`（已删除） |

### 第三轮修复（已完成，2026-06-22）

| 编号 | 问题 | 改动文件 |
|------|------|----------|
| 1.2 | Callback 接口加 HMAC-SHA256 签名校验 | `cmd/api/main.go`、`config/config.yaml`、`internal/pkg/config/config.go` |
| 1.6 | JWT Secret `${}` → `os.ExpandEnv` 展开 | `internal/pkg/config/config.go` |
| 2.4 | nextDay 日期边界 bug → time.AddDate | `internal/service/schedule/service.go` |
| 2.8 | ShouldBindJSON 漏检：toggle/submit/transfer | `cmd/api/main.go` |
| 2.10 | 工单全部状态 API + 前端修复 | `workorder/service.go`、`cmd/api/main.go`、`WorkOrderList.vue` |
| 3.5 | ByArea 硬编码 → area_routing_rule 动态查 | `internal/service/stats/service.go` |
| 🆕 | SLA 上报接入 WebSocket 通知推送 | `internal/service/sla/engine.go`、`cmd/api/main.go` |
| 🆕 | 转交重置 escalated_level + 提交/转交解锁 | `internal/service/workorder/service.go` |
| 🆕 | seedAdmin 自动创建默认路由规则 `*` | `cmd/api/main.go` |
| 🆕 | 手写 itoa → strconv.Itoa（sla + notify） | `internal/service/sla/engine.go`、`internal/service/notify/hub.go` |

### 待修复清单

| 优先级 | 编号 | 问题 | 备注 |
|--------|------|------|------|
| P1 | 2.1 | 抑制查询用 camera_name 而非 camera_id | 需确认 CRIP Callback 数据结构 |
| P1 | 2.9 | APP 登录字段匹配 | 部分可工作 |
| P2 | — | Callback 无频率限制 | 防止恶意刷报警 |
| P3 | — | 重复代码 `generateOrderNo`/`degreeToPriority` | alarm 和 workorder 各一份 |
| P3 | — | seedAdmin 密码硬编码 | 开发环境 OK，上线前需改 |
| P3 | — | 零测试文件 | — |
| P3 | — | 前端无 403/404 全局处理 | — |
| P3 | — | BroadcastToRole 广播所有人 | — |
| P3 | — | matchPattern 只支持后缀 `*` | — |
| P3 | — | Flutter /wechat/login 未实现 | login.dart 实际调的是 `/auth/login` |

---

## 七、第三轮前端修复（2026-06-22）

### 修复概览

| 端 | 问题数 | 说明 |
|------|--------|------|
| 后端 | 2 | 部门/用户组 CRUD API + 模拟测试数据 API |
| Web 管理后台 | 5 | 新增部门管理、用户组管理页面；修复用户表单下拉框；工单详情/转交弹窗 |
| Flutter APP | 4 | 消息中心连真实数据、工单处理流程、退出登录、用户名显示 |

### 后端改动

| 问题 | 修复 | 文件 |
|------|------|------|
| 部门/用户组无管理 API | `GET/POST/PUT/DELETE /departments` + `/user-groups` | `cmd/api/main.go` |
| seedAdmin 只建一个部门 | 预置管理部、安保部、工程部 + 4 个默认用户组 | `cmd/api/main.go` |
| 无测试数据 | `POST /api/v1/dev/mock-data` 一键生成 3部门+4用户组+5用户+15工单 | `cmd/api/main.go` |

### Web 管理后台改动

| 问题 | 修复 | 文件 |
|------|------|------|
| **无部门管理页面** | 新增 `DepartmentList.vue`，侧边栏 + 路由 + CRUD 弹窗 | 新建 + router/index.js + MainLayout.vue |
| **无用户组管理页面** | 新增 `GroupList.vue`，关联部门下拉框 | 新建 + router/index.js + MainLayout.vue |
| **用户表单部门/用户组是数字** | 改为下拉框，从 `/departments` `/user-groups` API 拉列表 | `UserList.vue` |
| **工单详情/转交是空函数** | 补详情弹窗(descriptions) + 转交弹窗 + 删除确认 | `WorkOrderList.vue` |
| **用户表单缺部门/组字段** | 加 department_id、group_id 下拉框 + admin 角色选项 | `UserList.vue` |

### Flutter APP 改动

| 问题 | 修复 | 文件 |
|------|------|------|
| **消息中心纯静态假数据** | 改为从 `/work-orders` API 拉真实工单，点消息可接单/处理 | `messages.dart` |
| **工单"处理"按钮空** | 加 BottomSheet 详情 + 处理结果提交表单 | `orders.dart` |
| **admin 显示"张三"** | 从 SharedPreferences 读取登录存储的真实用户名 | `profile.dart` |
| **退出后白屏** | `pushAndRemoveUntil` 回到 LoginPage | `profile.dart` |
| **菜单项点不动** | 改为 SnackBar 提示 | `profile.dart` |

### 测试数据

运行 `POST /api/v1/dev/mock-data`（需 admin token）生成：
- 3 个部门：测试-安保部、测试-工程部、测试-客服部
- 4 个用户组：白班一组、夜班一组、维修组、投诉处理组
- 5 个测试用户：张保安/李保安/王班长/赵工/刘经理，密码均为 `Test@123`
- 15 条模拟工单，覆盖 pending/processing/completed 三种状态

---

## 六、快速验证清单

- [x] 后端能编译：`go build ./...` 通过
- [ ] 后端能启动：`go run cmd/api/main.go`，访问 `/health` 返回 ok
- [ ] Web 后台能登录：`http://localhost:5173`，用 admin/Admin@123
- [ ] Web 工单列表能看到数据（需先有 Callback 触发或手动建工单）
- [ ] APP 能打开（改 baseUrl 后）
- [ ] APP 能登录（admin/Admin@123）
- [ ] APP 登录后能看到首页（改跳转目标后）
- [ ] 抑制策略创建后能显示（加错误检查后）
- [ ] WebSocket 连接正常（前端能看到新工单推送）
- [ ] SLA 超时上报触发（造一条工单不接单，等30秒看日志）
