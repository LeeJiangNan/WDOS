# WDOS 开发实施手册

> **版本**：v2.0
> **更新时间**：2026-06-23
> **状态**：核心功能已完成，进入调试优化阶段
> **部署位置**：昭阳（100.107.124.26）WSL2 Ubuntu 24.04

---

## 开发进度

```
总进度: ████████████░░░░ 85%

阶段 1  基础设施           ████████████ ✅ 完成
阶段 2  P0 核心业务         ████████████ ✅ 完成
阶段 3  P0 前端             ████████████ ✅ 完成
阶段 4  P1 增强             ██████████░░ ✅ 大部分完成
阶段 5  P2 优化             ░░░░░░░░░░░░ ⏳
```

---

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 + Redis 7 |
| 存储 | MinIO（图片/附件） |
| Web 后台 | Vue 3 + Element Plus + Vite |
| 移动端 | Flutter Web（HTML 渲染器） |
| 部署 | Docker Compose（MySQL/Redis/MinIO）|
| 回调中继 | 腾讯云 SCF + COS + 自研 Poller |

---

## 项目结构

```
WDOS/
├── cmd/api/main.go           # 入口 + 路由注册 + 种子数据
├── internal/
│   ├── model/                # 13 张数据表模型
│   ├── service/
│   │   ├── alarm/            # CRIP 回调处理（去重+图片+画框+路由+抑制+工单生成）
│   │   ├── auth/             # 认证（JWT + 明文密码）
│   │   ├── notify/           # WebSocket 推送
│   │   ├── route/            # 区域路由引擎（通配符匹配）
│   │   ├── schedule/         # 排班管理
│   │   ├── sla/              # SLA 超时引擎
│   │   ├── stats/            # 统计（角色过滤）
│   │   └── workorder/        # 工单 CRUD + 流转
│   ├── pkg/                  # 配置、日志、JWT
│   └── repository/           # MySQL/Redis/MinIO 连接
├── web/                      # Vue 3 管理后台
├── miniapp/                  # Flutter APP
├── scf/                      # 腾讯云 SCF 函数（callback 缓冲）
├── tools/poller/             # COS 轮询拉取程序
├── deploy/                   # Docker Compose
└── docs/                     # 设计文档
```

---

## 数据库表（13 张）

| 表 | 说明 |
|----|------|
| crip_alarm_raw | CRIP 原始报警存储 |
| work_order | 工单主表 |
| work_order_log | 工单操作日志 |
| work_order_template | 工单模板 |
| suppression_rule | 报警抑制规则 |
| area_routing_rule | 区域路由规则（部门工单配置） |
| **algorithm_routing_rule** | 算法工单配置 |
| sla_escalation_policy | SLA 上报策略 |
| staff_schedule | 人员排班 |
| users | 用户 |
| departments | 部门 |
| user_groups | 用户组 |
| workflow_definition | 工作流定义 |

---

## API 路由一览

| 分组 | 路由 | 方法 | 认证 |
|------|------|------|------|
| 健康检查 | /health | GET | 公开 |
| 认证 | /api/v1/auth/login | POST | 公开 |
| 认证 | /api/v1/auth/refresh | POST | JWT |
| Callback | /api/v1/callback/crip | POST | 公开 |
| Callback | /api/v1/callback/crip/compensate | POST | 公开 |
| 工单 | /api/v1/work-orders | GET | JWT |
| 工单 | /api/v1/work-orders/pending | GET | JWT |
| 工单 | /api/v1/work-orders/processing | GET | JWT |
| 工单 | /api/v1/work-orders/completed | GET | JWT |
| 工单 | /api/v1/work-orders/{id}/accept | POST | JWT |
| 工单 | /api/v1/work-orders/{id}/submit | POST | JWT |
| 工单 | /api/v1/work-orders/{id}/transfer | POST | JWT |
| 工单 | /api/v1/work-orders/{id} | DELETE | JWT |
| 模板 | /api/v1/work-order-templates | CRUD | JWT |
| 部门配置 | /api/v1/area-routing-rules | CRUD | JWT(admin) |
| 算法配置 | /api/v1/algorithm-routing-rules | CRUD | JWT(admin) |
| 部门 | /api/v1/departments | CRUD | JWT(admin) |
| 用户组 | /api/v1/user-groups | CRUD | JWT(admin) |
| 用户 | /api/v1/users | CRUD | JWT(admin) |
| 下属 | /api/v1/users/subordinates | GET | JWT(领导) |
| 排班 | /api/v1/schedules | GET/POST | JWT |
| 排班 | /api/v1/schedules/import-excel | POST | JWT(admin) |
| 统计 | /api/v1/stats/my-overview | GET | JWT |
| 统计 | /api/v1/stats/by-algorithm | GET | JWT |
| 统计 | /api/v1/stats/by-area | GET | JWT |
| 上传 | /api/v1/upload | POST | JWT |
| 文件 | /api/v1/minio/:bucket/*object | GET | 公开 |
| 开发 | /api/v1/dev/mock-data | POST | JWT(admin) |

---

## 当前数据状态

| 数据 | 数量 |
|------|------|
| 部门 | 4（管理部/安保部/物业部/工程部） |
| 用户组 | 3（安保白班/安保夜班/物业维修组） |
| 用户 | 8 |
| 区域路由规则 | 10 条 |
| 算法配置规则 | 8 条 |
| 排班记录 | 34 条（7天） |

---

## 部署信息

### 服务地址

| 服务 | 地址 |
|------|------|
| Web 管理后台 | http://100.107.124.26:5173 |
| Flutter APP | http://100.107.124.26:3000 |
| API 后端 | http://100.107.124.26:9090 |
| MinIO 控制台 | http://100.107.124.26:9001 |
| 局域网 APP | http://192.168.20.14:3000 |

### 登录账号

| 用户 | 密码 | 角色 | 部门 |
|------|------|------|------|
| admin | Admin@123 | 管理员 | 管理部 |
| 张安保 | Test@123 | 一线人员 | 安保部 |
| 李安保 | Test@123 | 一线人员 | 安保部 |
| 王班长 | Test@123 | 领班 | 安保部 |
| 刘经理 | Test@123 | 经理 | 安保部 |
| 赵工 | Test@123 | 一线人员 | 物业部 |
| 陈安保 | Test@123 | 一线人员 | 安保部 |

### CRIP Callback 地址

```
https://1446142760-f8s8ebtud7.ap-guangzhou.tencentscf.com
```

---

## 关键决策

1. 密码明文存储（调试阶段）
2. Flutter Web 使用 HTML 渲染器（手机兼容性）
3. 班组暂不限制，权限按部门开放
4. 路由匹配支持前缀/后缀/包含三种模式
5. 领导无日常消息，只收 SLA 超时通知
6. WSL2 vmIdleTimeout=-1 防进程被杀

---

## 待办

| 优先级 | 事项 |
|--------|------|
| P1 | WSL2 服务稳定性（守护脚本） |
| P1 | 排班：多天查看、手动调整、用户名显示 |
| P1 | 抑制策略 Web 管理页 |
| P2 | Flutter 倒计时 |
| P2 | Dashboard ECharts |
| P2 | WebSocket 前端接入 |
| P3 | 测试文件 |
| P3 | 前端错误全局处理 |
