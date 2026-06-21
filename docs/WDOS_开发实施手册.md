# WDOS 开发实施手册

> **版本**：v1.0
> **日期**：2026-06-21
> **状态**：开发阶段启动
> **GitHub**：https://github.com/LeeJiangNan/WDOS

---

## 目录

- [一、项目概述](#一项目概述)
- [二、环境信息](#二环境信息)
- [三、技术架构总览](#三技术架构总览)
- [四、开发阶段总览](#四开发阶段总览)
- [五、阶段 1：基础设施（2 天）](#五阶段-1基础设施2-天)
- [六、阶段 2：P0 核心业务（4 周）](#六阶段-2p0-核心业务4-周)
- [七、阶段 3：P0 前端（2 周）](#七阶段-3p0-前端2-周)
- [八、阶段 4：P1 增强（4 周）](#八阶段-4p1-增强4-周)
- [九、阶段 5：测试验收（2 周）](#九阶段-5测试验收2-周)
- [十、阶段 6：上线运维（1 周）](#十阶段-6上线运维1-周)
- [十一、代码规范](#十一代码规范)
- [十二、参考资料索引](#十二参考资料索引)

---

## 一、项目概述

### 1.1 项目定位

WDOS（Work-order Dispatch & Orchestration System）是外置于鲲云 CRIP 人工智能推理平台的商场工单系统。CRIP 产生 AI 视频分析报警后，通过 HTTP Callback 推送给 WDOS，WDOS 自动生成工单并推送到微信小程序，一线人员接单处理，形成「报警 → 派单 → 处理 → 归档」闭环。

### 1.2 当前状态

```
✅ 产品设计阶段全部完成:
  - 架构设计方案 v2.3
  - 65 个接口 Swagger 定义（15 章）
  - 业务流程闭环图（25 个问题全部确认）
  - 权限矩阵总表（8 个问题全部确认）
  - 接口对账表（65/65 全部定义）
  - 小程序交互原型
  - 管理后台交互原型

✅ 开发环境就绪:
  - GitHub 仓库已初始化
  - Go 1.22 + Docker + MySQL + Redis + MinIO 已部署
  - 项目骨架已推送

⏳ 待进行:
  - 代码实现（阶段 1-4）
  - 测试验收（阶段 5）
  - 上线运维（阶段 6）
```

### 1.3 关键依赖

| 依赖 | 类型 | 状态 |
|------|------|:--:|
| CRIP v3.14.0+ | 外部系统 | ⚠️ 需确认测试环境可用 |
| CRIP OpenAPI 凭证 | 外部凭证 | ⚠️ 需获取 app_id/secret |
| 微信小程序 AppID | 外部平台 | ⚠️ 需注册 |
| 微信小程序 AppSecret | 外部凭证 | ⚠️ 需获取 |

---

## 二、环境信息

### 2.1 开发机

| 项目 | 信息 |
|------|------|
| 机型 | MacBook Air M5 |
| 系统 | macOS Darwin 25.3.0 |
| Shell | zsh |
| Go | 1.22.12 darwin/arm64 |
| Docker | 29.4.1 |
| Node | 22.22.2 |

### 2.2 本地服务

| 服务 | 端口 | 账号 | 密码 |
|------|:--:|------|------|
| WDOS API | 8080 | — | — |
| MySQL | 3307 | wdos | wdos123 |
| Redis | 6380 | — | — |
| MinIO API | 9000 | minioadmin | minioadmin123 |
| MinIO 控制台 | 9001 | minioadmin | minioadmin123 |

### 2.3 常用命令

```bash
# 启动基础设施
make db-up

# 停止基础设施
make db-down

# 运行 API
make run

# 编译
make build

# 测试
make test

# 生成 Swagger
make docs

# 一键启动（基础设施 + API）
make dev
```

---

## 三、技术架构总览

```
┌──────────────────────────────────────────────────────────────┐
│                      前端展示层                               │
│  ┌────────────────────┐  ┌────────────────────────────┐     │
│  │  Vue 3 管理后台     │  │  微信小程序 (Taro + React)  │     │
│  │  Element Plus       │  │                            │     │
│  │  ECharts 5          │  │                            │     │
│  └────────┬───────────┘  └──────────────┬─────────────┘     │
│           │          HTTPS + JWT        │                    │
├───────────┼─────────────────────────────┼────────────────────┤
│           ▼                             ▼                    │
│                      接入层                                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Gin HTTP Router  +  JWT 中间件  +  Swagger          │   │
│  │  Callback 接收器 (/api/v1/callback/crip)             │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
├─────────────────────────┼────────────────────────────────────┤
│                         ▼                                    │
│                    业务服务层                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │ 报警处理  │ │ 工单管理  │ │ SLA 引擎  │ │ 工作流引擎    │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │ 抑制锁定  │ │ 区域路由  │ │ 通知推送  │ │ 排班管理      │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│                         │                                    │
├─────────────────────────┼────────────────────────────────────┤
│                         ▼                                    │
│                    数据存储层                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │  MySQL 8 │ │  Redis 7 │ │  MinIO   │ │ Elasticsearch │   │
│  │  业务数据 │ │ 缓存/队列 │ │ 图片存储  │ │  日志/搜索    │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## 四、开发阶段总览

```
阶段 1          阶段 2          阶段 3          阶段 4        阶段 5      阶段 6
基础设施        P0 核心业务     P0 前端         P1 增强       测试验收    上线运维
(2天)           (4周)           (2周)           (4周)         (2周)       (1周)
    │               │               │               │            │          │
    ▼               ▼               ▼               ▼            ▼          ▼
程序能启动      闭环能跑通      人能操作        体验完整      质量过关    线上运行
```

### 开发优先级原则

| 优先级 | 定义 | 判断标准 |
|:--:|------|----------|
| **P0** | 必须做 | 没有它核心业务流程跑不通 |
| **P1** | 应该做 | 没有它系统能用但体验差 |
| **P2** | 可以后面做 | 锦上添花，不影响主流程 |

### 完整功能清单

```
P0（第一批发版，13 周）
├── 后端 12 项（见阶段 2）
├── 管理后台 5 项（见阶段 3.1）
└── 小程序 6 项（见阶段 3.2）

P1（第二批发版，4 周）
├── 后端 7 项（见阶段 4）
├── 管理后台 6 项
└── 小程序 4 项

P2（第三批发版，按需）
└── 高级报表、工作流设计器、审计日志等
```

---

## 五、阶段 1：基础设施（2 天）

### 目标

`make run` 不报错，程序启动成功，能连上 MySQL/Redis/MinIO。

### 5.1 补齐配置加载

```
文件: internal/pkg/config/config.go

任务:
□ 定义 Config 结构体（对应 config.yaml 的字段）
□ 实现 Load(path) 函数
□ 环境变量注入（${VAR_NAME} 自动替换）
```

```go
// internal/pkg/config/config.go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    CRIP     CRIPConfig     `mapstructure:"crip"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    MinIO    MinIOConfig    `mapstructure:"minio"`
    Wechat   WechatConfig   `mapstructure:"wechat"`
    JWT      JWTConfig      `mapstructure:"jwt"`
    SLA      SLAConfig      `mapstructure:"sla"`
}

type ServerConfig struct {
    Port string `mapstructure:"port"`
    Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Name     string `mapstructure:"name"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
}

func (d DatabaseConfig) DSN() string {
    return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True",
        d.User, d.Password, d.Host, d.Port, d.Name)
}

// ... 其他结构体定义

func Load(path string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(path)
    v.AutomaticEnv() // 自动读取环境变量
    if err := v.ReadInConfig(); err != nil {
        return nil, err
    }
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### 5.2 补齐日志初始化

```
文件: internal/pkg/logger/logger.go

任务:
□ 初始化 zap 结构化日志
□ 根据 config.Server.Mode 切换 debug/release 级别
```

### 5.3 补齐数据库连接

```
文件: internal/repository/mysql/mysql.go

任务:
□ GORM 连接 MySQL
□ 自动建表（AutoMigrate）
□ 连接池配置
```

```go
// internal/repository/mysql/mysql.go
package mysql

import (
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "github.com/LeeJiangNan/WDOS/internal/model"
)

func Connect(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    // 自动建表
    db.AutoMigrate(
        &model.CRIPAlarmRaw{},
        &model.WorkOrder{},
        &model.WorkOrderLog{},
        // ... 其他表
    )
    return db, nil
}
```

### 5.4 补齐 Redis 连接

```
文件: internal/repository/redis/redis.go

任务:
□ go-redis 连接 Redis
□ 封装常用操作（SetNX 去重、Stream 发布订阅）
```

### 5.5 补齐 MinIO 连接

```
文件: internal/repository/minio/minio.go

任务:
□ minio-go 连接 MinIO
□ 封装文件上传（PutObject）
□ 确保 bucket 存在
```

### 5.6 模型定义

```
目录: internal/model/

任务:
□ 把设计文档第 7 节 8 张表转成 Go struct
□ gorm 标签（column、type、comment、index）
□ json 标签
```

```go
// internal/model/crip_alarm_raw.go
package model

import "time"

type CRIPAlarmRaw struct {
    ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
    SnowflakeID     string    `gorm:"uniqueIndex;size:64;not null" json:"snowflake_id"`
    CameraID        int       `gorm:"index:idx_camera_algo" json:"camera_id"`
    CameraUUID      string    `gorm:"size:64" json:"camera_uuid"`
    CameraName      string    `gorm:"size:100" json:"camera_name"`
    CameraGroup     string    `gorm:"type:json" json:"camera_group"`
    AlgorithmID     int       `gorm:"index:idx_camera_algo" json:"algorithm_id"`
    AlgorithmName   string    `gorm:"size:50" json:"algorithm_name"`
    AlgorithmNameEN string    `gorm:"size:100" json:"algorithm_name_en"`
    Degree          int       `json:"degree"`
    AlarmPicURL     string    `gorm:"size:500" json:"alarm_pic_url"`
    VideoURL        string    `gorm:"size:500" json:"video_url"`
    GPS             string    `gorm:"size:50" json:"gps"`
    RawJSON         string    `gorm:"type:json" json:"raw_json"`
    AlarmTimestamp  time.Time `gorm:"index:idx_alarm_time" json:"alarm_timestamp"`
    ReceivedAt      time.Time `gorm:"autoCreateTime" json:"received_at"`
    Source          string    `gorm:"type:enum('callback','compensation');default:'callback'" json:"source"`
    Suppressed      bool      `gorm:"default:false" json:"suppressed"`
    SuppressedByID  *uint64   `json:"suppressed_by_order_id"`
}

func (CRIPAlarmRaw) TableName() string {
    return "crip_alarm_raw"
}
```

**所有需要创建的模型文件：**

| 文件 | 对应表 |
|------|--------|
| `model/crip_alarm_raw.go` | crip_alarm_raw |
| `model/work_order.go` | work_order |
| `model/work_order_log.go` | work_order_log |
| `model/workflow_definition.go` | workflow_definition |
| `model/work_order_template.go` | work_order_template |
| `model/suppression_rule.go` | suppression_rule |
| `model/area_routing_rule.go` | area_routing_rule |
| `model/sla_escalation_policy.go` | sla_escalation_policy |
| `model/staff_schedule.go` | staff_schedule |
| `model/user.go` | users （用户表） |
| `model/department.go` | departments |
| `model/user_group.go` | user_groups |

### 5.7 阶段 1 检查清单

```
□ make run 不报错
□ 程序打印 "WDOS API 服务启动, 端口: 8080"
□ MySQL 中 8 张表已自动创建
□ Redis 连接正常
□ MinIO bucket "wdos" 已创建
□ curl http://localhost:8080/health 返回 200
```

---

## 六、阶段 2：P0 核心业务（4 周）

### 目标

CRIP Callback → 工单生成 → 小程序接单 → 处理 → 完成，整条链路跑通。

### 第 1 周：数据能进来

#### 6.1 Callback 接收器 ⭐ 最重要

```
文件: internal/transport/callback/handler.go
接口: POST /api/v1/callback/crip

任务:
□ 接收 CRIP JSON body
□ GIN 绑定 + 验证
□ Redis SETNX snowflake_id（去重）
□ 如果已存在 → 返回 { action: "ignored" }
□ 如果不存在:
  - 下载 alarm_pic_url 图片 → 存入 MinIO
  - raw_json 全量写入 MySQL
  - 关键字段提取到索引列
  - 写入 Redis Stream "alarm:raw"
  - 返回 { action: "created" } 或 { action: "suppressed" }
□ 整个处理 < 100ms，超时返回 200 但记录告警日志
```

**验收测试（curl）：**

```bash
curl -X POST http://localhost:8080/api/v1/callback/crip \
  -H "Content-Type: application/json" \
  -d '{
    "snowflake_id": "test-001",
    "camera_name": "测试摄像头",
    "algorithm_name": "行人闯入",
    "alarm_pic_url": "http://example.com/test.jpg",
    "timestamp": "2026-06-21 10:00:00"
  }'
```

#### 6.2 审计响应包装

```
文件: pkg/response/response.go

任务:
□ 统一 JSON 响应格式 { code, message, data }
□ 预定义错误码常量
```

#### 6.3 手动补偿接口

```
文件: internal/service/alarm/compensate.go
接口: POST /api/v1/admin/compensate

任务:
□ 管理员在后台点按钮触发
□ 调 CRIP POST /mc/v1/authenticate 获取 token
□ 调 CRIP POST /cb/v1/log/search 分页拉取报警日志
□ 逐条 snowflake_id 去重
□ 补漏的报警走正常工单生成流程
□ 返回 { total_fetched, newly_added, skipped }
```

#### 6.4 认证模块

```
文件: internal/service/auth/
接口: POST /api/v1/auth/wechat/login
      POST /api/v1/auth/login
      POST /api/v1/auth/refresh

任务:
□ 微信 code → openid → 查用户表 → 生成 JWT
□ Web 用户名密码登录 → 生成 JWT
□ JWT 中间件（Gin middleware，解析 + 鉴权）
□ Token 刷新
```

**JWT Payload：**

```json
{
  "user_id": 1001,
  "role": "handler",
  "department_id": 3,
  "group_id": 10,
  "exp": 1718956800
}
```

#### 6.5 心跳检测

```
文件: internal/service/alarm/heartbeat.go

任务:
□ Redis 计数器 "wdos:callback:count"
□ 每次收到 Callback → INCR
□ cron 每 5 分钟检查一次：
  - 计数器 > 0 → 正常，清零
  - 计数器 = 0 → 告警，管理后台首页标黄
```

### 第 2 周：工单能跑通

#### 6.6 工单模板管理

```
文件: internal/transport/http/template.go
接口: GET/POST/PUT /api/v1/templates
      POST /api/v1/templates/:id/toggle

任务:
□ 模板 CRUD
□ 表单组件配置存储（JSON 列）
□ 启用/停用
```

#### 6.7 工单生成引擎

```
文件: internal/service/workorder/generator.go

任务:
□ 消费 Redis Stream "alarm:raw"
□ 查抑制规则 → 判断是否抑制
□ 查区域路由 → 确定处理部门和人员
□ 创建 work_order 记录
□ 发布 WebSocket 通知
□ 启动 SLA 倒计时（Redis TTL）
```

#### 6.8 工单流转

```
文件: internal/transport/http/workorder.go
接口: GET  /api/v1/work-orders/pending
      GET  /api/v1/work-orders/processing
      GET  /api/v1/work-orders/completed
      GET  /api/v1/work-orders/:id
      POST /api/v1/work-orders/:id/accept
      POST /api/v1/work-orders/:id/submit
      POST /api/v1/work-orders/:id/transfer
      POST /api/v1/work-orders/:id/unlock

任务:
□ 待接单列表（角色限定范围）
□ 接单（独占锁，Redis SETNX）
□ 提交处理（含图片上传，multipart/form-data）
□ 转交（回到待接单，通知新处理人）
□ 解锁（管理员手动）
□ 操作日志（每次状态变更写 work_order_log）
```

**工单状态机：**

```
                  ┌──────────┐
                  │  pending  │ ← 报警生成 / 转交回来
                  └────┬─────┘
                       │ accept
                       ▼
                  ┌──────────┐
           ┌──────│processing│──────┐
           │      └────┬─────┘      │
           │ transfer  │  submit     │ transfer
           ▼           ▼            ▼
     ┌──────────┐ ┌──────────┐
     │ pending  │ │completed │
     └──────────┘ └──────────┘
```

#### 6.9 报警抑制

```
文件: internal/service/suppress/engine.go

任务:
□ 消费 Redis Stream "alarm:raw"
□ 查是否有同 camera_id + algorithm_id 的未处理工单
□ 有 → 抑制，duplicate_count += 1，更新截图
□ 无 → 创建新工单
□ 锁定判断（可选）→ 点位已锁 → 仅记录 raw_alarm
```

### 第 3-4 周：流程闭环

#### 6.10 SLA 超时引擎

```
文件: internal/service/sla/engine.go

任务:
□ cron 每秒扫描所有 pending/processing 的工单
□ 计算超时时间:
  - pending 工单: now - created_at > 阈值 → 上报
  - processing 工单: now - accepted_at > 阈值 → 上报
□ 上报逻辑:
  - L1 → 通知组长
  - L2 → 通知经理 + 触发锁定
  - L3 → 通知总监
□ 超时后仍可接单/处理（不倒计时结束就关单）
```

#### 6.11 通知推送

```
文件: internal/service/notify/

任务:
□ WebSocket 推送（工单中心实时更新）
□ 微信订阅消息（新工单、超时提醒）
□ 短信（紧急上报，可选）
```

#### 6.12 排班基础

```
文件: internal/transport/http/schedule.go
接口: GET  /api/v1/schedules
      POST /api/v1/schedules

任务:
□ 查看排班（按日期 + 部门）
□ 手动设置/修改排班
```

### 阶段 2 检查清单

```
□ curl 模拟 Callback → 数据库有 raw_alarm 记录
□ curl 模拟 Callback → 相同 snowflake_id 第二次调用返回 "ignored"
□ 管理后台能创建/编辑工单模板
□ Callback 生成工单 → 待接单列表能看到
□ 小程序接单 → 状态变为 processing
□ 小程序提交处理 → 状态变为 completed
□ 接单超时 30s → 能收到超时通知
□ 处理超时 150s → 能收到超时通知
□ 重复报警 → 抑制成功，duplicate_count 累加
```

---

## 七、阶段 3：P0 前端（2 周）

可以与阶段 2 第 3-4 周并行开发。

### 7.1 管理后台

```
目录: web/

□ Vue 3 + Element Plus 项目初始化     npm create vue@latest
□ 路由配置（登录 + 各页面）
□ API 封装（axios + JWT 拦截器）
```

#### 页面清单

| 页面 | 路由 | 对应原型 | P0/P1 |
|------|------|----------|:--:|
| 登录 | `/login` | 登录页 | P0 |
| 首页大屏 | `/dashboard` | 首页大屏 | P1 |
| 工单模板管理 | `/templates` | 模板列表 + 创建弹窗 | P0 |
| 工单数据管理 | `/work-orders` | 数据列表 + 筛选 | P0 |
| 抑制策略配置 | `/suppression` | 策略列表 + 创建弹窗 | P0 |
| 用户管理 | `/users` | 用户列表 + 创建弹窗 | P0 |
| 权限配置 | `/permissions` | 权限矩阵表格 | P0 |
| 区域路由配置 | `/routing` | 路由列表 | P1 |
| SLA 策略配置 | `/sla` | SLA 列表 + 创建弹窗 | P1 |
| 部门管理 | `/departments` | 部门列表 | P1 |
| 用户组管理 | `/groups` | 用户组列表 | P1 |
| 排班日历 | `/schedule` | 排班视图 | P1 |
| 排班导入 | `/schedule/import` | 上传 + 预览 | P1 |
| 统计报表 | `/stats` | 统计图表 | P1 |

### 7.2 微信小程序

```
目录: miniapp/
框架: Taro + React + TypeScript

初始化:
  npm install -g @tarojs/cli
  taro init miniapp
  # 选择 React + TypeScript 模板
```

#### 页面清单

| 页面 | 路由 | 对应原型 | P0/P1 |
|------|------|----------|:--:|
| 工作台 | `pages/home/index` | 首页（按角色展示不同内容） | P0 |
| 工单中心 | `pages/orders/index` | 待接单/待处理/已完成 + 筛选 | P0 |
| 工单详情 | `pages/orders/detail` | 报警信息 + 截图 + 处理 | P0 |
| 处理工单 | `pages/orders/process` | 填表单 + 拍照 + 签名 | P0 |
| 消息 | `pages/messages/index` | 通知列表 | P0 |
| 我的 | `pages/my/index` | 个人统计 + 排班查看 + 设置 | P0 |
| 统计报表 | `pages/stats/index` | 图表统计 | P1 |
| 排班查看 | `pages/schedule/index` | 个人排班日历 | P1 |

#### 关键 API 封装

```typescript
// miniapp/src/services/api.ts
const BASE_URL = 'https://wdos.yourmall.com/api/v1'

export async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const token = Taro.getStorageSync('access_token')
  const res = await Taro.request({
    url: BASE_URL + url,
    header: { Authorization: `Bearer ${token}` },
    ...options
  })
  if (res.data.code !== 0) {
    throw new Error(res.data.message)
  }
  return res.data.data
}
```

---

## 八、阶段 4：P1 增强（4 周）

P0 闭环稳定后启动。

### 8.1 后端增强

```
□ 锁定机制（internal/service/suppress/lock.go）
    - 点位锁定 + 自动/手动解锁
    - 锁定期间报警记录

□ 区域路由引擎（internal/service/route/engine.go）
    - camera_group 匹配 → 部门 → 处理人
    - 算法覆盖路由

□ 多角色看板（internal/service/stats/dashboard.go）
    - 领班/经理/总监各自首页数据
    - 下钻查询（总监→经理→领班→一线）

□ SLA 策略可配置（internal/transport/http/sla.go）
    - 管理后台修改超时阈值
    - 通知方式可配（微信/短信/电话）

□ 排班 Excel 导入（internal/transport/http/schedule.go）
    - POST /api/v1/schedules/import (multipart)
    - Excel 解析 → 冲突检测 → 预览 → 确认导入

□ 报警图片画框（internal/service/alarm/annotator.go）
    - object_list.rect → 在图片上绘制检测框
    - 存入 MinIO /alarms/anno/{snowflake_id}.jpg

□ 统计接口全部实现（internal/service/stats/）
    - daily-overview, by-algorithm, by-area, distribution, trend, user-ranking
```

### 8.2 前端增强

```
□ 管理后台: 区域路由配置页、SLA 策略配置页、排班日历、排班导入、统计报表、数据大屏
□ 小程序: 统计报表页、图片全屏缩放、两级筛选（算法+区域）
```

---

## 九、阶段 5：测试验收（2 周）

### 9.1 测试类型

| 类型 | 内容 | 负责人 |
|------|------|:--:|
| **接口测试** | 每个 API 至少 1 条正常 + 2 条异常用例 | 开发者 |
| **端到端测试** | CRIP → Callback → 工单 → 接单 → 处理的完整链路 | 开发者 + 产品 |
| **异常测试** | 断网、重启、并发、弱网 | 开发者 |
| **UAT** | 真人试用 | 产品（你） |

### 9.2 异常测试清单

```
□ WDOS 重启 → Redis Stream 消费是否恢复
□ CRIP Callback 停止 → 心跳是否告警
□ 手动补偿是否正常补漏
□ MySQL 连接断开 → GORM 是否自动重连
□ Redis 满 → 报警数据是否丢失
□ 同一工单两人同时接单 → 是否只有一个成功
□ 微信小程序弱网 → 接单操作是否有重试
□ 大图片上传 → 是否触发超时
□ 并发 Callback 100条/秒 → 去重是否正常
```

### 9.3 UAT 流程

```
1. 准备 UAT 环境（独立部署，用测试数据库）
2. 找 2-3 名真实安保人员
3. 给他们小程序，观察以下操作:
   □ 能否独立登录
   □ 看到待接单通知后能否找到工单
   □ 能否接单 → 处理 → 填写表单 → 拍照 → 签名 → 提交
   □ 转交操作是否流畅
   □ 消息通知是否及时
4. 记录所有操作不顺畅的地方
5. 汇总 → 评估严重程度 → 修 bug → 再测
```

### 9.4 阶段 5 检查清单

```
□ 所有接口正常场景通过
□ 所有接口异常场景有合理的错误提示
□ 端到端链路 5 次不出错
□ 所有异常场景有兜底方案
□ UAT 反馈已处理
□ 没有 P0 级别的 bug
```

---

## 十、阶段 6：上线运维（1 周）

### 10.1 上线前准备

```
□ 生产环境服务器就绪
□ 生产数据库初始化（执行 migrations）
□ 管理员账号创建
□ CRIP Callback 地址配置为生产地址
□ 微信小程序提交审核（预留 1-7 天）
□ 监控告警配置
□ 回滚方案准备
```

### 10.2 上线步骤

```
1. 生产环境部署
   scp docker-compose.yml config.yaml user@prod-server:/opt/wdos/
   ssh user@prod-server
   cd /opt/wdos && docker-compose up -d

2. 数据库迁移
   go run ./cmd/migrate --env=production

3. 管理后台部署
   cd web && npm run build
   scp -r dist/* user@prod-server:/opt/wdos/web/

4. 小程序提交审核
   cd miniapp && taro build --type weapp
   登录 mp.weixin.qq.com → 上传代码 → 提交审核

5. CRIP 对接
   CRIP 后台 → Callback 推送地址 → https://wdos-prod.xxx.com/api/v1/callback/crip

6. 灰度上线
   先让安保部用 1-2 天，无问题全量

7. 监控值守
   第一周 7×24 关注错误日志和告警
```

### 10.3 回滚方案

```
如果上线后出现重大问题:
  1. CRIP Callback 地址切回旧系统（如果有）
  2. K8s: kubectl rollout undo deployment/wdos-api
  3. Docker: docker-compose down && 切回旧版本 docker-compose.yml && docker-compose up -d
  4. 数据库不用回滚（新表不影响旧数据）
```

---

## 十一、代码规范

### 11.1 命名规范

```
Go:
  包名         小写，单数（alarm, workorder, suppress）
  文件名       小写 + 下划线（generator.go, lock_engine.go）
  导出函数     大写开头（NewService, GetByID）
  非导出函数    小写开头（validateCamera, buildQuery）
  接口         单方法用 -er 后缀（Reader, Writer）

数据库:
  表名         小写 + 下划线（work_order, suppression_rule）
  字段名       小写 + 下划线（camera_id, created_at）
  索引名       idx_表名_字段（idx_work_order_status）

API:
  路径         小写 + 连字符（/api/v1/work-orders）
  参数         小写 + 下划线（start_date, user_id）
```

### 11.2 目录规范

```
每个 service 模块固定三个文件:
  service.go    → 接口定义 + 构造函数 NewXxxService()
  xxx.go        → 核心业务逻辑
  xxx_test.go   → 单元测试
```

### 11.3 Git 提交规范

```bash
feat: Callback 接收器
fix: 修复接单超时未上报的 bug
docs: 更新 API 文档
refactor: 重构工单状态机
test: 补充 SLA 引擎单元测试
chore: 更新依赖版本
```

### 11.4 代码审查清单

```
□ 是否遵循目录规范
□ 是否有单元测试
□ 错误处理是否完整
□ 是否有 SQL 注入风险
□ 敏感信息是否硬编码（必须走配置文件 + 环境变量）
□ 日志是否够用（关键操作有 Info，异常有 Error）
□ API 响应格式是否统一
```

---

## 十二、参考资料索引

| 文档 | 路径 | 什么时候看 |
|------|------|-----------|
| 架构设计方案 v2.3 | `docs/design/WDOS商场AI工单系统架构设计方案v2.0.md` | 不确定业务逻辑时 |
| API 接口定义 | `docs/WDOS_API接口定义文档.md` | 写接口时，65 个全部有 Swagger |
| 业务流程闭环图 | `docs/WDOS_业务流程闭环图.md` | 理不清某条流程时 |
| 权限矩阵总表 | `docs/WDOS_权限矩阵总表.md` | 写鉴权中间件时 |
| 接口对账表 | `docs/WDOS_接口对账表.md` | 确认接口覆盖 |
| 设计流程 v1.1 | `docs/WDOS_B端产品标准设计流程v1.1.md` | 排期参考 |
| 小程序 UI 原型 | `工单系统/WDOS_小程序原型.html` | 前端开发时对照 |
| 管理后台 UI 原型 | `工单系统/WDOS_管理后台原型.html` | 前端开发时对照 |
| CRIP Callback 文档 | `工单系统/crip文档/` | 对接 Callback 数据格式 |
| CRIP OpenAPI 文档 | `工单系统/crip文档/` | 手动补偿接口调 CRIP |
| GitHub 仓库 | `https://github.com/LeeJiangNan/WDOS` | 代码 |

---

> **文档状态**：v1.0，产品经理交付开发团队的完整施工手册。
>
> **下一步**：找到 Go 开发者 → 从阶段 1 开始逐项实现 → 每周检查进度对照本文档。
