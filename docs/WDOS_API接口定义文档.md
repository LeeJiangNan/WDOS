# WDOS API 接口定义文档

> **版本**：v2.0
> **更新时间**：2026-06-23
> **Base URL**：`http://100.107.124.26:9090/api/v1`（开发环境）
> **认证方式**：Bearer Token（JWT）

---

## 目录

- [1. 认证接口](#1-认证接口)
- [2. Callback 接收](#2-callback-接收)
- [3. 工单中心](#3-工单中心)
- [4. 工单模板管理](#4-工单模板管理)
- [5. 区域路由规则（部门工单配置）](#5-区域路由规则部门工单配置)
- [6. 算法工单配置](#6-算法工单配置)
- [7. 人员管理](#7-人员管理)
- [8. 部门管理](#8-部门管理)
- [9. 用户组管理](#9-用户组管理)
- [10. 排班管理](#10-排班管理)
- [11. 统计接口](#11-统计接口)
- [12. 文件管理](#12-文件管理)
- [13. 通用规范](#13-通用规范)

---

## 1. 认证接口

### 1.1 Web 登录（用户名/姓名/手机号）

```
POST /api/v1/auth/login
```

**请求体：**
```json
{ "username": "admin", "password": "Admin@123" }
```

**响应：**
```json
{
  "code": 0, "message": "success",
  "data": {
    "access_token": "eyJhbGci...",
    "expires_in": 604800,
    "user": { "id": 1, "username": "admin", "name": "admin", "role": "admin", "department_id": 1 }
  }
}
```

### 1.2 Token 刷新

```
POST /api/v1/auth/refresh
Header: Authorization: Bearer {token}
```

---

## 2. Callback 接收

### 2.1 CRIP 报警推送

```
POST /api/v1/callback/crip
Content-Type: application/json
```

**完整字段参见开发调试记录 §1.2**

**响应：**
```json
{
  "action": "created",       // created | suppressed | ignored
  "work_order_id": 123,      // 生成的工单 ID
  "suppressed": false,       // 是否被抑制
  "reason": "成功生成工单"
}
```

### 2.2 CRIP 报警补偿

```
POST /api/v1/callback/crip/compensate
```

---

## 3. 工单中心

### 3.1 工单列表

```
GET /api/v1/work-orders                    # 全部状态
GET /api/v1/work-orders/pending            # 待接单
GET /api/v1/work-orders/processing         # 处理中
GET /api/v1/work-orders/completed          # 已完成
```

**参数**：`?view_as={user_id}`（领导代理查看）

**响应：**
```json
{
  "code": 0, "message": "success",
  "data": {
    "total": 36,
    "list": [
      {
        "id": 52, "order_no": "WD-20260623-xxx", "title": "人员入侵 - 1F1028",
        "status": "pending", "priority": "high", "degree": 3,
        "department_id": 5, "department_name": "安保部",
        "assignee_id": 6, "assignee_name": "安保白班",
        "accepter_name": null, "camera_name": "1F1028商户门口",
        "algorithm_name": "人员入侵识别", "alarm_pic_url": "/minio/wdos/alarms/raw/xxx.jpg",
        "duplicate_count": 1, "created_at": "2026-06-23 15:35:29"
      }
    ]
  }
}
```

### 3.2 接单

```
POST /api/v1/work-orders/{id}/accept
```

### 3.3 提交处理

```
POST /api/v1/work-orders/{id}/submit
{
  "resolution": "已处理完毕",
  "form_data": "{\"attachments\":[\"/minio/wdos/attachments/xxx.jpg\"]}",
  "proof_images": "/minio/wdos/attachments/xxx.jpg"
}
```

### 3.4 转交

```
POST /api/v1/work-orders/{id}/transfer
{ "transfer_to_user_id": 9, "transfer_to_user_name": "张安保", "reason": "转交原因" }
```

### 3.5 删除

```
DELETE /api/v1/work-orders/{id}
```

---

## 4. 工单模板管理

```
GET    /api/v1/work-order-templates         # 列表
POST   /api/v1/work-order-templates         # 新增
PUT    /api/v1/work-order-templates/{id}    # 更新
DELETE /api/v1/work-order-templates/{id}    # 删除
POST   /api/v1/work-order-templates/{id}/toggle  # 启用/停用
```

---

## 5. 区域路由规则（部门工单配置）

```
GET    /api/v1/area-routing-rules           # 列表（按优先级排序）
POST   /api/v1/area-routing-rules           # 新增
PUT    /api/v1/area-routing-rules/{id}      # 更新
DELETE /api/v1/area-routing-rules/{id}      # 删除
```

**字段说明：**
| 字段 | 类型 | 说明 |
|------|------|------|
| camera_group_pattern | string | 匹配模式，支持 `B1*`/`*机房`/`*扶梯*` |
| area_name | string | 区域显示名称 |
| department_id | int | 分配部门 ID |
| handler_group_id | int | 处理班组 ID（暂不用，置 0） |
| priority | int | 优先级，越大越先匹配 |
| is_active | bool | 是否启用 |

---

## 6. 算法工单配置

```
GET    /api/v1/algorithm-routing-rules      # 列表
POST   /api/v1/algorithm-routing-rules      # 新增
PUT    /api/v1/algorithm-routing-rules/{id} # 更新
DELETE /api/v1/algorithm-routing-rules/{id} # 删除
```

**字段说明：**
| 字段 | 类型 | 说明 |
|------|------|------|
| algorithm_pattern | string | 算法名称匹配（与 CRIP algorithm_name 一致） |
| display_name | string | 显示别名 |
| department_id | int | 分配部门 ID |
| category | string | 分类（消防/安防/其他） |
| priority | int | 优先级 |
| is_active | bool | 是否启用 |

---

## 7. 人员管理

### 7.1 用户 CRUD（需 admin）

```
GET    /api/v1/users                      # 列表（支持 ?role=handler 过滤）
POST   /api/v1/users                      # 新增
PUT    /api/v1/users/{id}                 # 更新
```

### 7.2 下属列表（领导功能）

```
GET /api/v1/users/subordinates
```

**权限**：supervisor/manager 返回同部门下属，admin/director 返回所有人

---

## 8. 部门管理

```
GET    /api/v1/departments                 # 列表
POST   /api/v1/departments                 # 新增
PUT    /api/v1/departments/{id}            # 更新
DELETE /api/v1/departments/{id}            # 删除
```

---

## 9. 用户组管理

```
GET    /api/v1/user-groups                 # 列表（支持 ?department_id= 过滤）
POST   /api/v1/user-groups                 # 新增
PUT    /api/v1/user-groups/{id}            # 更新
DELETE /api/v1/user-groups/{id}            # 删除
```

---

## 10. 排班管理

### 10.1 按日期查排班

```
GET /api/v1/schedules?date=2026-06-23
```

**响应：** 按班次类型分组 `{ "day": [...], "night": [...] }`

### 10.2 Excel 导入

```
POST /api/v1/schedules/import-excel
Content-Type: multipart/form-data
```

### 10.3 设置排班

```
POST /api/v1/schedules
{ "user_id": 9, "shift_date": "2026-06-23", "shift_type": "day", "area": "B1/B2" }
```

---

## 11. 统计接口

### 11.1 我的概览

```
GET /api/v1/stats/my-overview
```

**响应（角色过滤）：**
```json
{
  "date": "2026-06-23", "total_orders": 5,
  "pending_orders": 2, "processing_orders": 1, "completed_orders": 2,
  "overtime_orders": 1, "completion_rate": 0.4
}
```

### 11.2 算法分布 / 区域分布 / 耗时分布 / 趋势 / 人员排行

```
GET /api/v1/stats/by-algorithm?date=2026-06-23
GET /api/v1/stats/by-area?date=2026-06-23
GET /api/v1/stats/process-time?date=2026-06-23
GET /api/v1/stats/trend?days=7
GET /api/v1/stats/user-ranking?date=2026-06-23
```

---

## 12. 文件管理

### 12.1 上传

```
POST /api/v1/upload
Content-Type: multipart/form-data
Field: file
```

### 12.2 读取 MinIO 文件（公开）

```
GET /api/v1/minio/{bucket}/{object}
```

---

## 13. 通用规范

### 13.1 响应格式

```json
{
  "code": 0,           // 0=成功, 40100=未登录, 40300=无权限, 40400=未找到
  "message": "success",
  "data": {}
}
```

### 13.2 鉴权

所有业务 API 需携带 `Authorization: Bearer {jwt_token}`

### 13.3 时间格式

所有时间字段统一为 `"2026-06-23 15:35:29"`（北京时间）

### 13.4 角色可见范围

| 角色 | 数据范围 |
|------|---------|
| handler | 本部门 |
| supervisor | 本部门 |
| manager | 本部门 |
| admin / director | 全量 |

### 13.5 分页

`?page=1&page_size=20`，默认 page_size=20，最大 100
