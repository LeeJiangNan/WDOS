# WDOS - 商场 AI 工单调度与编排系统

基于鲲云 CRIP 人工智能推理平台的商场工单系统，将 AI 视频分析报警自动转化为可追踪的工单流程。

## 技术栈

| 组件 | 选型 |
|------|------|
| 后端 | Go 1.22+ / Gin / GORM |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7 |
| 对象存储 | MinIO |
| 管理后台 | Vue 3 + Element Plus + ECharts |
| 移动端 | Flutter APP（主力）+ 微信小程序 Taro（兜底） |

---

## 快速开始

### 前提条件

- Docker Desktop（已安装）
- Go 1.22+（已安装）
- Node.js 18+（管理后台需要）

### 第一步：启动基础设施

```bash
cd WDOS
make db-up
```

自动下载并启动 MySQL + Redis + MinIO 三个容器。

### 第二步：编译并启动后端

```bash
# 编译（macOS 需要 CGO_ENABLED=0）
CGO_ENABLED=0 go build -o wdos-server ./cmd/api

# 启动
./wdos-server
```

或者一键运行：

```bash
CGO_ENABLED=0 go run ./cmd/api
```

### 第三步：启动管理后台

```bash
cd web
npm install
npm run dev
```

浏览器打开 `http://localhost:3000`

### 第四步：启动 Flutter APP（可选）

```bash
cd miniapp
flutter pub get
flutter run
```

---

## 访问地址

| 服务 | 地址 | 账号/密码 |
|------|------|------|
| API 服务 | http://localhost:8080 | — |
| 健康检查 | http://localhost:8080/health | — |
| WebSocket | ws://localhost:8080/ws/notifications | — |
| 管理后台 | http://localhost:3000 | admin / Admin@123 |
| MinIO 控制台 | http://localhost:9001 | minioadmin / minioadmin123 |

---

## 命令行速查

```bash
make db-up       # 启动 MySQL + Redis + MinIO
make db-down     # 停止
make run         # 启动 Go API
make build       # 编译二进制
go test ./...    # 运行测试
```

---

## 核心 API 一览

### 无需认证

```
POST /api/v1/callback/crip          CRIP 报警回调
```

### 认证

```
POST /api/v1/auth/login             管理后台登录 { username, password }
POST /api/v1/auth/wechat/login      微信登录 { code }
POST /api/v1/auth/refresh           Token 刷新
```

### 工单模板

```
GET    /api/v1/templates            模板列表 (?status=active)
GET    /api/v1/templates/:id        模板详情
POST   /api/v1/templates            创建模板
PUT    /api/v1/templates/:id        编辑模板
POST   /api/v1/templates/:id/toggle 启用/停用 { is_active }
```

### 工单中心

```
GET    /api/v1/work-orders/pending      待接单列表
GET    /api/v1/work-orders/processing   待处理列表
GET    /api/v1/work-orders/completed    已完成列表
GET    /api/v1/work-orders/:id          工单详情
POST   /api/v1/work-orders/:id/accept   接单
POST   /api/v1/work-orders/:id/submit   提交处理 { resolution, form_data }
POST   /api/v1/work-orders/:id/transfer 转交 { transfer_to_user_id, reason }
```

### 管理功能

```
POST   /api/v1/admin/compensate         手动补偿 { start_time, end_time }
GET    /api/v1/suppression-rules        抑制策略列表
POST   /api/v1/suppression-rules        创建策略
GET    /api/v1/routing-rules            区域路由列表
POST   /api/v1/routing-rules            创建路由
GET    /api/v1/users                    用户列表
POST   /api/v1/users                    新增用户
GET    /api/v1/schedules                排班查询 (?date=YYYY-MM-DD)
POST   /api/v1/schedules                设置排班
```

### 统计

```
GET    /api/v1/stats/daily-overview             每日概览
GET    /api/v1/stats/by-algorithm                算法分布
GET    /api/v1/stats/by-area                     区域统计
GET    /api/v1/stats/process-time-distribution   耗时分布
GET    /api/v1/stats/trend                       近N天趋势
GET    /api/v1/stats/user-ranking                人员排行
```

---

## 测试 Callback 模拟

```bash
# 模拟 CRIP 报警推送
curl -X POST http://localhost:8080/api/v1/callback/crip \
  -H 'Content-Type: application/json' \
  -d '{
    "snowflake_id": "test-001",
    "camera_id": 10,
    "camera_name": "B1停车场C区3号通道",
    "algorithm_id": 4,
    "algorithm_name": "行人闯入",
    "degree": "3",
    "timestamp": "2026-06-21 10:00:00",
    "camera_group": ["parking_B1"]
  }'

# 完整工作流测试
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin@123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

# 查看待接单
curl http://localhost:8080/api/v1/work-orders/pending \
  -H "Authorization: Bearer $TOKEN"

# 接单
curl -X POST http://localhost:8080/api/v1/work-orders/1/accept \
  -H "Authorization: Bearer $TOKEN"

# 处理完成
curl -X POST http://localhost:8080/api/v1/work-orders/1/submit \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"resolution":"已处理","form_data":"{}"}'
```

---

## 项目结构

```
cmd/api/main.go             入口 + 路由注册
internal/
  model/                    12 个 GORM 模型
  pkg/
    config/                 配置加载 (viper)
    jwt/                    JWT 生成/验证
    logger/                 zap 日志
  repository/
    mysql/                  GORM 连接
    redis/                  go-redis 连接
    minio/                  MinIO 连接
  service/
    alarm/                  报警处理 + 补偿
    auth/                   认证 (微信/Web/Token)
    workorder/              工单服务 + 模板
    sla/                    SLA 超时引擎
    notify/                 WebSocket 通知
    route/                  区域路由引擎
    schedule/               排班服务
    stats/                  统计服务
  transport/http/           JWT 中间件
pkg/response/               统一 API 响应
config/config.yaml           配置文件
deploy/docker-compose.yml   基础设施
web/                         Vue 3 管理后台
miniapp/                     Flutter APP
docs/                        设计文档副本
```

---

## 开发进度

```
████████████████████ 全部完成

后端 (Go)       26 源文件  ~3,000 行  35+ API ✅
管理后台 (Vue)   13 文件     ~600 行   8 页面 ✅
移动端 (Flutter) 8 文件     ~500 行   5 页面 ✅
设计文档         17 份     6 文件夹 完整 PRD ✅
```

---

## 设计文档

完整的产品设计文档在 `工单系统/` 文件夹下（与代码仓库同级）：

| 文档 | 路径 | 用途 |
|------|------|------|
| 架构设计方案 v2.4 | `产品设计/WDOS商场AI工单系统架构设计方案v2.0.md` | 技术架构、部署策略 |
| 业务流程闭环图 | `产品设计/WDOS_业务流程闭环图.md` | 全角色流程 |
| 权限矩阵总表 | `产品设计/WDOS_权限矩阵总表.md` | 角色 × 模块权限 |
| 接口对账表 | `产品设计/WDOS_接口对账表.md` | 65 接口 × 客户端 |
| 关键决策记录 | `产品设计/WDOS_关键决策记录.md` | 24 条关键决策 |
| API 接口定义 | `接口文档/WDOS_API接口定义文档.md` | 65 个接口 Swagger |
| 开发实施手册 | `开发规划/WDOS_开发实施手册.md` | 开发流程 + 排期 |
| 小程序原型 | `原型/WDOS_小程序原型.html` | 浏览器打开可交互 |
| 管理后台原型 | `原型/WDOS_管理后台原型.html` | 浏览器打开可交互 |

---

## 分支策略

- `main` — 稳定版本
- `develop` — 开发主线
- `feature/*` — 功能分支

## 许可

内部项目，鲲云科技。
