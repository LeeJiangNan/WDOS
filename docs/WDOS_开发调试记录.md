# WDOS 开发调试记录

> 时间：2026-06-22 ~ 2026-06-23  
> 环境：昭阳（100.107.124.26）Windows 11 + WSL2 Ubuntu 24.04  
> 后端：Go 1.22 + Gin + GORM + MySQL 8.0 + Redis 7 + MinIO  
> 目标机器公网 IP：14.153.163.211（路由器未做端口映射）  
> 局域网 IP：192.168.20.14

---

## 一、CRIP Callback 集成

### 1.1 端点

```
POST /api/v1/callback/crip
Content-Type: application/json
```

### 1.2 CRIP 原生 JSON 格式（已全部兼容）

```json
{
  "snowflake_id": "1768287987212271600",   // 字符串或数字均可（FlexString 兼容）
  "camera_id": 10,                          // int
  "camera_uuid": "1b6883bf...",
  "camera_name": "Camera 1",
  "camera_group": [],                       // 空数组或 null 均可
  "camera_types": [1],
  "algorithm_id": 4,                        // int
  "algorithm_name": "行人闯入",
  "algorithm_name_en": "CR_PERSON_INVASION",
  "degree": "3",                            // 字符串或数字均可（FlexString 兼容）
  "alarm_pic_url": "",                      // ⚠️ 必须为空或真实URL，example.com会超时
  "alarm_pic_data": "/9j/...base64...",     // base64 图片数据，URL 不可达时兜底
  "timestamp": "2026-06-22 21:00:00",       // ⚠️ 必须是真实时间
  "gps": "50.85045,4.34878",
  "members": [
    {
      "user_id": 200,                       // 数字或字符串均可
      "user_name": "John Doe",
      "role": "Black",
      "score": 95,
      "tag": "test"
    }
  ],
  "result_data": [
    {
      "algorithm_name": "行人闯入",
      "degree": "3",
      "task_id": 4,
      "result_data": {
        "task_id": 4,
        "task_result": {
          "class_id": 1,
          "object_list": [
            {
              "class_id": 0,
              "rect": {"x": 224, "y": 605, "width": 206, "height": 349},
              "score": 0.85
            }
          ]
        }
      }
    }
  ]
}
```

### 1.3 Callback 处理流程

```
CRIP 推送 → ① 签名校验（callback_secret为空则跳过）
           → ② snowflake_id 去重（Redis SETNX，24小时）
           → ③ 获取图片：alarm_pic_url 下载 → 失败则 base64 解码
           → ④ 检测框绘制（result_data.object_list.rect → 红色矩形框）
           → ⑤ 上传到 MinIO
           → ⑥ 存储原始报警到 MySQL
           → ⑦ 区域路由（camera_group → 路由规则 → department_id）
           → ⑧ 抑制检查（同摄像头+同算法有未处理工单→追加计数）
           → ⑨ 生成工单（含部门名称、指派人）
           → ⑩ WebSocket 推送通知 + SLA 超时计时
```

### 1.4 返回结果

| action | 含义 |
|--------|------|
| `created` | 新工单生成 |
| `suppressed` | 抑制（已有未处理工单，追加重复计数） |
| `ignored` | 忽略（snowflake_id 重复） |

### 1.5 网络状态

| 方式 | 状态 |
|------|------|
| Tailscale (100.107.124.26:9090) | ✅ Mac 可访问 |
| 公网直连 (14.153.163.211:9090) | ❌ 路由器未做端口映射 |
| localhost.run 隧道 | ❌ 国外 IP，CRIP 不可达 |
| serveo.net 隧道 | ❌ 国外 IP，CRIP 不可达 |
| frp 免费服务器 | ❌ DNS 无法解析 |
| **SCF+COS+Poller（最终方案）** | ✅ **已上线** |

### 1.6 最终方案：SCF + COS + Poller（2026-06-23 部署）

```
CRIP 公网 → SCF(广州) → COS(广州) → 昭阳 Poller → WDOS
```

| 组件 | 地址 | 说明 |
|------|------|------|
| SCF 函数 URL | `https://1446142760-f8s8ebtud7.ap-guangzhou.tencentscf.com` | CRIP 推送地址 |
| COS 存储桶 | `wdos-callback-1446142760` | 广州，私有读写 |
| Poller 服务 | 昭阳 WSL2 systemd | 每 10s 拉取，自动重启 |

---

## 二、用户指定的业务规则

### 2.1 登录方式
- 支持三种登录：**用户名** / **姓名** / **手机号**
- 所有用户密码**明文存储**（调试阶段）
- User 模型字段：`username`、`name`、`phone`、`password`、`role`、`department_id`、`group_id`

### 2.2 接单流程
1. 工单生成 → 通知一线人员
2. 一线人员可在 APP 接单
3. 超时30秒 → 通知领班（不强提醒一线）
4. 超时120秒 → 通知经理 + 触发锁定
5. 一线人员超时后仍可接单
6. 领班/经理可见部门所有工单

### 2.3 处理提交流程
1. 接单后弹出处理表单
2. 错误报警：选"错误报警"直接提交
3. 正常报警：填写处理说明 + 可上传附件（图片）
4. 提交后工单变为"已完成"，解除锁定

### 2.4 转交
- 从下拉框选择目标用户（同部门或上级）
- 转交后工单回到"待接单"状态
- 重置上报层级、解锁

### 2.5 区域路由规则
- B1/B2（除机房）→ 安保部
- 1F-5F（除机房）→ 物业部
- 所有机房 → 工程部
- test / 外广场 → 管理部
- 兜底规则 * → 管理部

### 2.6 算法工单配置
- `algorithm_name` → 部门分配规则
- 与部门工单配置平行，合在一起决定工单归属
- 处理方式：写结果 + 上传附件

### 2.7 排班
- 现有员工每周七天排班（白班/夜班/公休）
- Excel 导入自动创建用户（默认密码 123456）
- APP 端显示本周排班卡片

### 2.8 领导身份切换
- 领班/经理：查看同部门下级数据
- admin/director：查看所有人数据
- 领导平时无消息，只有 SLA 超时到对应等级才有提醒

### 2.9 工单可见性
- handler：本部门所有工单
- supervisor/manager：本部门所有工单
- admin/director：全量工单

---

## 三、后端改动记录

### 3.1 安全性
| 改动 | 文件 |
|------|------|
| HMAC-SHA256 签名校验（callback） | `cmd/api/main.go` |
| `${VAR}` 环境变量展开（`os.ExpandEnv`） | `internal/pkg/config/config.go` |
| JWT 中间件挂载所有业务路由 | `cmd/api/main.go` |
| CORS 调试模式允许所有来源 | `cmd/api/main.go` |

### 3.2 数据模型
| 改动 | 文件 |
|------|------|
| User 加 `username` 字段（唯一索引） | `internal/model/user.go` |
| WorkOrder 加 `assignee_name` 字段 | `internal/model/work_order.go` |
| AlgorithmRoutingRule 新表 | `internal/model/algorithm_routing_rule.go` |
| LocalTime 自定义时间类型（JSON 格式 "2006-01-02 15:04:05"） | `internal/model/types.go` |
| SuppressionRule JSON 字段改为 `*string`（可空） | `internal/model/suppression_rule.go` |
| FlexString 类型（兼容 JSON 数字/字符串） | `internal/model/callback.go` |
| CRIPMember.UserID / CRIPCallback.Degree 改为 FlexString | `internal/model/callback.go` |

### 3.3 密码
| 改动 | 文件 |
|------|------|
| bcrypt → 明文存储 | `auth/service.go`、`cmd/api/main.go` |
| 用户列表返回密码字段 | `cmd/api/main.go` |
| PUT 改为 `db.Save`（全字段更新） | `cmd/api/main.go` |

### 3.4 新增 API
| 路由 | 说明 |
|------|------|
| `GET/POST /departments` | 部门 CRUD |
| `PUT/DELETE /departments/:id` | 部门编辑/删除 |
| `GET/POST /user-groups` | 用户组 CRUD |
| `PUT/DELETE /user-groups/:id` | 用户组编辑/删除 |
| `GET/POST/PUT/DELETE /area-routing-rules[/:id]` | **区域路由规则 CRUD** |
| `GET/POST/PUT/DELETE /algorithm-routing-rules[/:id]` | **算法工单配置 CRUD** |
| `GET /users/subordinates` | **下属列表（领导功能）** |
| `GET /minio/*` | **MinIO 文件代理读取** |
| `POST /schedules/import-excel` | 排班 Excel 导入 |
| `GET /mock-image` | 模拟报警图片（640×480 带检测框） |
| `POST /upload` | 文件上传到 MinIO |
| `POST /dev/mock-data` | 一键生成测试数据 |

### 3.5 Bug 修复
| 问题 | 修复 |
|------|------|
| `ShouldBindJSON` 不检查错误 | toggle/submit/transfer 全部补齐 |
| `db.Updates` 零值不更新 | 用户 PUT 改为先查再 `db.Save` |
| nextDay 日期边界 bug | `time.AddDate(0,0,1)` 替换手写 |
| 密码修改不生效 | bcrypt→明文 + `db.Save` |
| 抑制策略 JSON 报错 | `string`→`*string` |
| SLA 引擎不上报通知 | 接入 notifyHub + escalation 推送 |
| 转交不重置状态 | escalated_level=0, is_locked=false |
| 工单全部状态 404 | 新增 `ListAll` + `GET /work-orders` |
| 排班统计硬编码 | 从 `area_routing_rule` 表动态查 |
| `itoa` 手写 | `strconv.Itoa` 替换 |
| CRIP degree 字段 int→string | FlexString 兼容 |
| 报警图片不可见 | alarm_pic_data base64 解码 + 画框 |
| 图片路径 404 | 新增 `/minio/` 文件代理路由 |
| handler 看不见部门工单 | 改为 department_id 过滤 |
| 工单生成无部门/指派人名 | 创建时查询 department/user_group 表 |
| `created_at` 显示 null | PUT 时不应提交 created_at |
| 时间格式不友好 | LocalTime 统一输出 "2006-01-02 15:04:05" |

### 3.6 架构增强
| 改动 | 说明 |
|------|------|
| 路由匹配引擎升级 | 支持后缀匹配（`*机房`）和包含匹配（`*扶梯*`） |
| 图片处理链路 | URL 下载 → base64 解码 → 检测框绘制 → MinIO |

---

## 四、前端改动记录

### 4.1 Web 管理后台（Vue 3）

| 页面 | 改动 |
|------|------|
| 用户管理 | username 列、密码列、部门下拉框、用户组下拉框 |
| 部门管理 | CRUD 弹窗 |
| 用户组管理 | 关联部门下拉框 |
| 工单数据 | 详情弹窗（含图片）、转交弹窗、删除确认 |
| 排班管理 | 日期选择器 + 今日排班表格 |
| **部门工单配置** | **新页面**：区域路由规则 CRUD，支持通配符匹配模式 |
| **算法工单配置** | **新页面**：算法名称→部门分配规则 CRUD |
| 侧边栏 | 新增 部门工单配置、算法工单配置 菜单 |

### 4.2 Flutter APP（Dart）

| 页面 | 改动 |
|------|------|
| **工作台** | 统计卡片（可点击跳转到工单中心对应 Tab）、工单列表（含状态标签、图片 thumbnail、等级颜色、重复次数）、筛选弹窗（按区域/算法） |
| **工单中心** | 三 Tab（待接单/处理中/已完成）、全屏详情页（接单确认、处理提交、转交下拉框、文件上传）、筛选弹窗 |
| **消息中心** | 真实工单列表、点击进入全屏详情页 |
| **我的** | 真实用户名、**身份切换（领导功能）**、**本周排班卡片**、数据统计弹窗（待接单/处理中/已完成/超时完成/超时未完成） |
| 登录 | 存用户信息到 SharedPreferences |
| Tab 导航 | 消息角标（待接单数量）、页面间跳转 |

### 4.3 特性
- 报警等级颜色（0灰/1蓝/2橙/3红/4紫）+ 罗马数字标签（0/Ⅰ/Ⅱ/Ⅲ/Ⅳ）
- 重复报警标签（"重复N次"橙色标签）
- 报警图片 thumbnail + 点击全屏可缩放（InteractiveViewer）
- 时间友好化（"3分钟前"/"2小时前"）
- 筛选器：按区域（部门工单配置 pattern）筛选 + 按算法名称筛选
- 错误报警/正常报警双通道提交
- 确认对话框（接单确认、提交确认）
- 文件上传（处理工单时可上传图片附件，已完成工单可查看）
- HTML 渲染器（手机浏览器兼容）
- 动态 API 地址适配（Tailscale/局域网自动切换）

---

## 五、部署信息

### 5.1 服务地址
| 服务 | 地址 | 说明 |
|------|------|------|
| Web 管理后台 | http://100.107.124.26:5173 | Vue 3 Vite dev server |
| Flutter APP | http://100.107.124.26:3000 | Python http.server |
| API 后端 | http://100.107.124.26:9090 | Go Gin |
| MinIO 控制台 | http://100.107.124.26:9001 | minioadmin/minioadmin123 |
| 局域网 Flutter | http://192.168.20.14:3000 | 手机同 WiFi 访问 |
| 健康检查 | http://100.107.124.26:9090/health | 返回 MySQL/Redis/MinIO 状态 |

### 5.2 登录账号
| 用户 | 密码 | 角色 | 部门 |
|------|------|------|------|
| admin | Admin@123 | 管理员 | 管理部 |
| 张安保 | Test@123 | 一线人员 | 安保部 |
| 李安保 | Test@123 | 一线人员 | 安保部 |
| 王班长 | Test@123 | 领班 | 安保部 |
| 刘经理 | Test@123 | 经理 | 安保部 |
| 赵工 | Test@123 | 一线人员 | 物业部 |
| 陈安保 | Test@123 | 一线人员 | 安保部 |
| gg | Test@123 | 一线人员 | 物业部 |

### 5.3 关键文件路径
```
昭阳 WSL2：/home/corerain/WDOS/
后端二进制：/home/corerain/wdos-server
Flutter APP：/home/corerain/flutter-app/
Web 管理后台：/home/corerain/WDOS/web/
```

### 5.4 重启命令
```bash
# 昭阳 WSL2 交互终端中
# 一键恢复所有服务
cd /home/corerain/WDOS && bash ~/wdos-init.sh

# 或手动：
pkill -f wdos-server
cd /home/corerain/WDOS
JWT_SECRET=wdos-dev-jwt-secret-2026 nohup /home/corerain/wdos-server > /home/corerain/wdos-server.log 2>&1 &

cd /home/corerain/WDOS/web && nohup npx vite --host 0.0.0.0 --port 5173 > /dev/null 2>&1 &
cd /home/corerain/flutter-app/web && nohup python3 -m http.server 3000 > /dev/null 2>&1 &
systemctl restart wdos-poller
```

---

## 六、当前系统状态 vs 原型/设计对比

### 6.1 已实现 ✅

| 功能 | 状态 |
|------|------|
| CRIP 回调接收 + 去重 + 存储 | ✅ |
| 报警图片 base64 解码 + 检测框绘制 | ✅ |
| 工单生成 + 抑制策略 | ✅ |
| 区域路由（部门工单配置）Web 管理 | ✅ |
| 算法工单配置 Web 管理 | ✅ |
| 部门/用户组/用户 CRUD Web 管理 | ✅ |
| 排班管理（Excel 导入 + 日期查看） | ✅ |
| Web 工单详情（含图片预览） | ✅ |
| Flutter 工作台（统计卡片 + 工单列表 + 筛选） | ✅ |
| Flutter 工单中心（三 Tab + 接单/处理/转交） | ✅ |
| Flutter 消息中心 | ✅ |
| Flutter 我的（排班 + 数据统计） | ✅ |
| Flutter 身份切换（领导视图） | ✅ |
| Flutter 附件上传 | ✅ |
| 确认对话框（接单/提交） | ✅ |
| SCF+COS+Poller Callback 中继 | ✅ |
| 时间格式统一（2026-06-23 15:35:29） | ✅ |
| 部门内全员可见工单 | ✅ |

### 6.2 未实现 / 差距 ⚠️

| 功能 | 原计划 | 当前状态 |
|------|--------|---------|
| **抑制策略 Web UI** | 算法+摄像头组合选择器 | ❌ 只有 DB 表，无 Web 管理页 |
| **SLA 策略 Web UI** | 超时等级配置 | ❌ 只有 DB 表，硬编码配置 |
| **排班多天查看** | 周视图/月视图 | ⚠️ 目前只有单日查看 |
| **排班手动调整** | Web 端直接编辑 | ❌ 只能 Excel 导入 |
| **Dashboard ECharts** | 图表统计大屏 | ❌ Web 首页只有文字统计 |
| **Flutter 倒计时** | 工单处理倒计时 | ❌ |
| **WebSocket 实时推送** | APP 实时收到新工单 | ⚠️ 后端已实现，前端未接入 |
| **手机端通知** | Push 通知 | ❌ |
| **测试文件** | 单元测试/集成测试 | ❌ 0 个测试 |
| **前端 403/404 全局处理** | 统一错误页面 | ❌ |
| **微信小程序** | 原生微信小程序 | ❌ 改成了 Flutter Web |
| **排班姓名显示** | 表格显示员工姓名 | ⚠️ 只显示 user_id |

### 6.3 与原型图差异

| 原型设计 | 实际实现 | 差异 |
|---------|---------|------|
| 微信小程序 | Flutter Web APP | 平台不同，功能等效 |
| PC Web + 移动端分离 | Web 管理后台 + Flutter APP | 架构不同但覆盖 |
| HTML 原型中的视频策略 | 未实现 | 原型阶段已决定去掉视频 |
| 工单详情弹窗 | 全屏页面 | HTML 渲染器兼容性修复 |

---

## 七、待办事项

| 优先级 | 事项 |
|--------|------|
| P0 | ~~**SCF+COS+Poller 部署**~~ ✅ |
| P0 | ~~**真实 CRIP 数据推送联调**~~ ✅ |
| P0 | ~~**图片+检测框**~~ ✅ |
| P0 | ~~**区域路由 + 算法配置**~~ ✅ |
| P0 | ~~**领导身份切换**~~ ✅ |
| P1 | WSL2 服务稳定性（写守护脚本，防掉线） |
| P1 | 排班管理：多天查看、手动调整、用户名显示 |
| P1 | 抑制策略 Web 管理页 |
| P2 | Flutter 倒计时显示 |
| P2 | Dashboard ECharts 图表 |
| P2 | WebSocket 前端接入（实时刷新） |
| P3 | 零测试文件 |
| P3 | 前端 403/404 全局处理 |
| P3 | 手机端 Push 通知 |

---

## 八、关键决策记录

1. **密码明文存储**：调试阶段，上线前需改回 bcrypt
2. **callback 签名校验可关闭**：`callback_secret: ""` 跳过
3. **vmIdleTimeout=-1**：防止 WSL2 空闲关机杀进程
4. **Tailscale 冲突**：WSL2 内的 Tailscale 抢回程路由，已关闭
5. **镜像源**：Go→goproxy.cn，npm→npmmirror.com，apt→aliyun
6. **Flutter HTML 渲染器**：CanvasKit 在手机浏览器不兼容，改为 HTML 渲染
7. **工单详情全屏**：AlertDialog 在 HTML 渲染器下灰屏，改为全屏页面
8. **班组暂不限制**：部门工单配置中的班组字段置 0，权限按部门开放
9. **区域路由匹配模式**：支持前缀（`B1*`）、后缀（`*机房`）、包含（`*扶梯*`）
10. **领导无日常消息**：领导只收 SLA 超时上报通知
11. **Windows cmd 管道问题**：SSH → wsl 时 `&&`、`&`、`|` 会被 cmd 拦截，所有多步骤命令必须写成脚本文件执行
12. **SCP 大文件不稳定**：23MB 后端二进制走 Tailscale SCP 可能超时，大文件部署时优先用昭阳本地终端
13. **Flutter API 响应解包层级**：axios 拦截器返回 `res.data`，前端取数据要区分 `res.data.list`（嵌套）和 `res.data`（直接数组）
14. **区域筛选用 routing pattern**：筛选器的区域选项从 `area_routing_rule` 表取 `camera_group_pattern`（如 `B1*`），匹配时去 `*` 做前缀比对
15. **手机浏览器 Flutter 兼容性**：CanvasKit 渲染器在 iOS Safari 等浏览器不显示文字，需改用 HTML 渲染器
