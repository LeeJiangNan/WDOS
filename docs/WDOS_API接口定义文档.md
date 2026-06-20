# WDOS API 接口定义文档

> **版本**：v1.0
> **Base URL**：`https://wdos.yourmall.com`
> **文档格式**：Swagger / OpenAPI 3.0

---

## 目录

- [1. 认证接口](#1-认证接口)
- [2. Callback 接收](#2-callback-接收)
- [3. 工单中心](#3-工单中心)
- [4. 工单模板管理](#4-工单模板管理)
- [5. 工单数据管理](#5-工单数据管理)
- [6. 报警抑制规则](#6-报警抑制规则)
- [7. 区域路由规则](#7-区域路由规则)
- [8. SLA 上报策略](#8-sla-上报策略)
- [9. 人员管理](#9-人员管理)
- [10. 排班管理](#10-排班管理)
- [11. 权限管理](#11-权限管理)
- [12. 统计接口](#12-统计接口)
- [13. WebSocket 通知](#13-websocket-通知)
- [14. 通知历史](#14-通知历史)
- [15. 通用规范](#15-通用规范)

---

## 1. 认证接口

### 1.1 微信小程序登录

```
POST /api/v1/auth/wechat/login
```

**请求体：**
```json
{
  "code": "081xwz0w3Qp6aX2YJ41w3OWkto3xwz0v"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 604800,
    "user": {
      "id": 1001,
      "name": "张三",
      "phone": "13800138000",
      "role": "handler",
      "department": "安保部"
    }
  }
}
```

### 1.2 Web 管理后台登录

```
POST /api/v1/auth/login
```

**请求体：**
```json
{
  "username": "admin",
  "password": "Admin@123"
}
```

**响应：** 同 1.1（不含微信绑定信息）

### 1.3 Token 刷新

access_token 有效期 7 天，过期前调用此接口刷新。写在 JWT 过期时间的前 1 天内调用即可避免重新登录。

```
POST /api/v1/auth/refresh
Authorization: Bearer <token>
```

**请求体：**（无，JWT 从 Header 读取）

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 604800
  }
}
```

**错误场景：**
```json
{ "code": 40100, "message": "token 已过期无法刷新，请重新登录", "data": null }
{ "code": 40101, "message": "用户已被禁用，无法刷新 token", "data": null }
```

---

## 2. Callback 接收

### 2.1 接收 CRIP 报警回调

```
POST /api/v1/callback/crip
Header: X-CRIP-Signature: <optional HMAC签名>
```

**请求体（CRIP 原文透传）：**
```json
{
  "snowflake_id": "1768287987212271616",
  "analysis_job_id": "ee6234ca5a7541dba61062d66ad82a8b",
  "timestamp": "2026-06-14 15:04:05",
  "camera_id": 10,
  "camera_uuid": "aa6234ca5a7541dba61062d66ad82a8b",
  "camera_name": "B1停车场C区3号通道",
  "camera_group": ["parking_B1"],
  "camera_types": [1],
  "channel_id": "34020000001310000001",
  "gps": "50.85045,4.34878",
  "stream_url": "rtsp://192.168.0.1:3554/live/2",
  "online_status": 1,
  "algorithm_id": 4,
  "algorithm_name": "行人闯入",
  "algorithm_name_en": "CR_PERSON_INVASION",
  "degree": "3",
  "alarm_pic_url": "http://crip.yourmall.com/files/alarm_pic.jpg",
  "alarm_pic_data": "base64...",
  "alarm_pic_name": "alarm_pic.jpg",
  "src_pic_url": "http://crip.yourmall.com/files/src_pic.jpg",
  "src_pic_data": "base64...",
  "src_pic_name": "src_pic.jpg",
  "video_url": "http://crip.yourmall.com/files/video.mp4",
  "video_name": "alarm_video.mp4",
  "image_width": 1920,
  "image_height": 1080,
  "extra": "{\"company\":\"万达广场\"}",
  "members": [
    {
      "user_id": "MSR_21",
      "user_name": "John Doe",
      "tag": "stranger",
      "score": 0.5,
      "photo": "http://crip.yourmall.com/files/face.jpg",
      "role": ""
    }
  ],
  "result_data": [
    {
      "algorithm_name": "行人闯入",
      "algorithm_en_name": "CR_PERSON_INVASION",
      "degree": "3",
      "task_id": 4,
      "result_data": {
        "task_id": 4,
        "task_result": {
          "class_id": 1,
          "extra_data": "[{\"cn_name\":\"人数\",\"name\":\"count\",\"show\":true,\"type\":\"float\",\"vals\":[3]}]",
          "object_list": [
            {
              "class_id": 0,
              "rect": {"x": 224, "y": 605, "width": 206, "height": 349},
              "score": 0.853873,
              "extra_data": "[{\"cn_name\":\"人头检测分数\",\"name\":\"head_det_score\",\"vals\":[0.93]}]"
            }
          ],
          "score": 0
        }
      }
    }
  ]
}
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "action": "suppressed",
    "work_order_id": 1523,
    "suppressed": true,
    "reason": "同摄像头同算法存在未处理工单"
  }
}
```

**action 枚举：**
| action | 说明 |
|--------|------|
| `created` | 成功创建新工单 |
| `suppressed` | 被抑制，追加到已有工单 |
| `locked` | 点位已锁定，仅记录不生成工单 |
| `ignored` | 重复雪花ID，已忽略 |

**响应码：**
| HTTP Code | 说明 |
|-----------|------|
| 200 | 处理成功（不管是否抑制都返回 200） |
| 400 | 请求体解析失败 |
| 500 | 内部错误 |

---

## 3. 工单中心

### 3.1 待接单列表

```
GET /api/v1/work-orders/pending?page=1&size=20&keyword=行人闯入&camera_name=B1
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 15,
    "page": 1,
    "size": 20,
    "list": [
      {
        "id": 1525,
        "order_no": "WD-20260614-01525",
        "title": "行人闯入 - B1停车场C区3号通道",
        "camera_name": "B1停车场C区3号通道",
        "algorithm_name": "行人闯入",
        "degree": 3,
        "priority": "high",
        "alarm_pic_url": "https://wdos.yourmall.com/files/minio/alarm_1525.jpg",
        "alarm_time": "2026-06-14 15:04:05",
        "duplicate_count": 3,
        "created_at": "2026-06-14 15:04:05",
        "sla_accept_deadline": "2026-06-14 15:04:35",
        "escalated_level": 0
      }
    ]
  }
}
```

### 3.2 待处理列表

```
GET /api/v1/work-orders/processing?page=1&size=20
Authorization: Bearer <token>
```

**响应：** 同 3.1 结构，增加 `accepted_at`、`sla_process_deadline`

### 3.3 已完成列表

```
GET /api/v1/work-orders/completed?page=1&size=20&start_date=2026-06-01&end_date=2026-06-14
Authorization: Bearer <token>
```

**响应：** 同 3.1 结构，增加 `completed_at`、`resolution`

### 3.4 工单详情

```
GET /api/v1/work-orders/:id
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1525,
    "order_no": "WD-20260614-01525",
    "template_id": 1,
    "title": "行人闯入 - B1停车场C区3号通道",
    "status": "pending",
    "priority": "high",
    "degree": 3,
    "department_name": "安保部",
    "assignee_group_name": "安保巡逻组",

    "camera_id": 10,
    "camera_uuid": "aa6234ca5a7541dba61062d66ad82a8b",
    "camera_name": "B1停车场C区3号通道",
    "camera_group": ["parking_B1"],
    "gps": "50.85045,4.34878",

    "algorithm_id": 4,
    "algorithm_name": "行人闯入",
    "algorithm_name_en": "CR_PERSON_INVASION",

    "alarm_pic_url": "https://wdos.yourmall.com/files/minio/alarm_1525.jpg",
    "src_pic_url": "https://wdos.yourmall.com/files/minio/src_1525.jpg",
    "video_url": "https://wdos.yourmall.com/files/minio/video_1525.mp4",

    "duplicate_count": 3,
    "suppressed_alarm_count": 0,
    "is_locked": false,

    "accepter": null,
    "assignee": null,
    "sla_accept_deadline": "2026-06-14 15:04:35",
    "sla_process_deadline": null,
    "escalated_level": 0,

    "form_data": null,
    "resolution": null,

    "alarm_time": "2026-06-14 15:04:05",
    "created_at": "2026-06-14 15:04:05",
    "accepted_at": null,
    "completed_at": null,

    "object_list": [
      {"class_id": 0, "rect": {"x": 224, "y": 605, "width": 206, "height": 349}, "score": 0.853873}
    ],
    "members": [
      {"user_id": "MSR_21", "user_name": "John Doe", "tag": "stranger", "score": 0.5}
    ],

    "logs": [
      {
        "id": 1,
        "action": "created",
        "operator_name": "系统",
        "comment": "CRIP自动生成工单",
        "created_at": "2026-06-14 15:04:05"
      },
      {
        "id": 2,
        "action": "suppressed",
        "operator_name": "系统",
        "comment": "第2次重复报警，已抑制",
        "created_at": "2026-06-14 15:04:10"
      },
      {
        "id": 3,
        "action": "suppressed",
        "operator_name": "系统",
        "comment": "第3次重复报警，已抑制",
        "created_at": "2026-06-14 15:04:14"
      }
    ]
  }
}
```

### 3.5 接单

```
POST /api/v1/work-orders/:id/accept
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1525,
    "status": "processing",
    "accepter": {"id": 1001, "name": "张三"},
    "accepted_at": "2026-06-14 15:04:20",
    "sla_process_deadline": "2026-06-14 15:06:50"
  }
}
```

**错误场景：**
```json
{
  "code": 40001,
  "message": "工单已被他人接单",
  "data": null
}
```

### 3.6 提交处理

```
POST /api/v1/work-orders/:id/submit
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**请求体（multipart/form-data）：**
| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `resolution` | string | ✅ | 处理结果描述 |
| `form_data` | string(JSON) | ✅ | 表单数据 |
| `proof_images[]` | file | ☐ | 现场照片（最多5张） |
| `proof_video` | file | ☐ | 现场视频 |

```json
{
  "resolution": "已确认是清洁工误入，已劝离。",
  "form_data": "{\"处理结果\":\"误报\",\"现场情况\":\"清洁工在通道内打扫卫生\"}"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1525,
    "status": "completed",
    "completed_at": "2026-06-14 15:05:30",
    "resolution": "已确认是清洁工误入，已劝离。",
    "proof_images": [
      "https://wdos.yourmall.com/files/minio/proof_1525_1.jpg",
      "https://wdos.yourmall.com/files/minio/proof_1525_2.jpg"
    ]
  }
}
```

### 3.7 转交工单

```
POST /api/v1/work-orders/:id/transfer
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "transfer_to_user_id": 1005,
  "reason": "此区域超出我的管辖范围"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1525,
    "status": "pending",
    "assignee_id": 1005,
    "accepter_id": null,
    "escalated_level": 1
  }
}
```

### 3.8 解除锁定

```
POST /api/v1/work-orders/:id/unlock
Authorization: Bearer <token>
```

**请求体（可选）：**
```json
{
  "reason": "已确认现场安全，手动解除"
}
```

---

## 4. 工单模板管理

### 4.1 模板列表

```
GET /api/v1/templates?status=active&page=1&size=20
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 5,
    "list": [
      {
        "id": 1,
        "name": "行人闯入处理工单",
        "description": "用于处理行人闯入禁区的报警",
        "flow_name": "标准派单处理",
        "form_schema": { "components": [] },
        "is_active": true,
        "created_at": "2026-06-01 10:00:00",
        "updated_at": "2026-06-10 14:30:00"
      }
    ]
  }
}
```

### 4.2 创建模板

```
POST /api/v1/templates
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "name": "烟雾报警处理工单",
  "description": "用于处理烟雾传感器的报警",
  "flow_id": 2,
  "form_schema": {
    "components": [
      {"type": "text", "label": "报警地点", "field_id": "camera_name", "mapping": "camera_name", "readonly": true},
      {"type": "text", "label": "报警类型", "field_id": "algorithm_name", "mapping": "algorithm_name", "readonly": true},
      {"type": "image", "label": "报警截图", "field_id": "alarm_pic", "mapping": "alarm_pic_url", "readonly": true},
      {"type": "select", "label": "处理结果", "field_id": "result", "options": ["真实火情", "误报", "测试", "其他"], "required": true},
      {"type": "textarea", "label": "处理说明", "field_id": "description", "required": true},
      {"type": "upload_image", "label": "现场照片", "field_id": "proof_images", "max": 5},
      {"type": "signature", "label": "处理人签名", "field_id": "signature", "required": true}
    ]
  },
  "handler_group_ids": [10, 11]
}
```

### 4.3 更新模板

```
PUT /api/v1/templates/:id
Authorization: Bearer <token>
```

请求体同 4.2。

### 4.4 启用/停用

```
POST /api/v1/templates/:id/toggle
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "is_active": false
}
```

---

## 5. 工单数据管理

### 5.1 全部工单列表（管理员）

```
GET /api/v1/admin/work-orders?status=pending&department_id=3&start_date=2026-06-01&end_date=2026-06-14&keyword=WD-&page=1&size=20
Authorization: Bearer <token>
```

### 5.2 导出工单

```
POST /api/v1/admin/work-orders/export
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "status": "completed",
  "start_date": "2026-06-01",
  "end_date": "2026-06-14",
  "department_id": 3,
  "format": "excel"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "导出任务已提交，完成后将通过站内信通知下载",
  "data": {
    "task_id": "export_20260614_001"
  }
}
```

### 5.3 管理员转交

```
POST /api/v1/admin/work-orders/:id/force-transfer
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "transfer_to_user_id": 1008,
  "reason": "原处理人休假"
}
```

### 5.4 管理员删除

```
DELETE /api/v1/admin/work-orders/:id
Authorization: Bearer <token>
```

---

## 6. 报警抑制规则

### 6.1 规则列表

```
GET /api/v1/suppression-rules?page=1&size=20
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 3,
    "list": [
      {
        "id": 1,
        "name": "停车场区域抑制策略",
        "camera_group_filter": ["parking_*"],
        "algorithm_ids": [4, 5],
        "suppress_enabled": true,
        "lock_enabled": true,
        "lock_after_seconds": 300,
        "lock_mode": "algo_only",
        "max_lock_seconds": 3600,
        "is_active": true
      }
    ]
  }
}
```

### 6.2 创建规则

```
POST /api/v1/suppression-rules
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "name": "设备区烟雾报警抑制",
  "camera_group_filter": ["equipment_*"],
  "algorithm_ids": [1],
  "suppress_enabled": true,
  "lock_enabled": true,
  "lock_after_seconds": 120,
  "lock_mode": "algo_only",
  "max_lock_seconds": 3600,
  "unlock_on_degree_up": true,
  "unlock_on_new_algo": true,
  "record_suppressed": true,
  "notify_on_lock": true,
  "summary_on_unlock": true
}
```

### 6.3 更新/删除

```
PUT  /api/v1/suppression-rules/:id
DELETE /api/v1/suppression-rules/:id
```

---

## 7. 区域路由规则

### 7.1 规则列表

```
GET /api/v1/routing-rules?page=1&size=20
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "camera_group_pattern": "parking_*",
        "area_name": "停车场",
        "department_name": "安保部",
        "handler_group_name": "安保巡逻组",
        "backup_group_name": "安保班长",
        "priority": 10,
        "is_active": true
      }
    ]
  }
}
```

### 7.2 CRUD

```
POST   /api/v1/routing-rules
PUT    /api/v1/routing-rules/:id
DELETE /api/v1/routing-rules/:id
```

---

## 8. SLA 上报策略

### 8.1 策略列表

```
GET /api/v1/sla-policies?page=1&size=20
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "商场常规报警SLA",
        "accept_l1_seconds": 30,
        "accept_l1_group_name": "安保班长",
        "accept_l2_seconds": 120,
        "accept_l2_group_name": "安保经理",
        "accept_l3_seconds": 300,
        "accept_l3_group_name": "商场总监",
        "process_l1_seconds": 150,
        "process_l1_group_name": "安保班长",
        "process_l2_seconds": 300,
        "process_l2_group_name": "安保经理",
        "process_l3_seconds": 600,
        "process_l3_group_name": "商场总监",
        "notify_channels": ["wechat", "sms"],
        "is_active": true
      }
    ]
  }
}
```

### 8.2 CRUD

```
POST   /api/v1/sla-policies
PUT    /api/v1/sla-policies/:id
DELETE /api/v1/sla-policies/:id
```

---

## 9. 人员管理

### 9.1 用户列表

```
GET /api/v1/users?department_id=3&role=handler&keyword=张三&page=1&size=20
Authorization: Bearer <token>
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 25,
    "list": [
      {
        "id": 1001,
        "name": "张三",
        "phone": "13800138000",
        "role": "handler",
        "role_name": "一线保安",
        "department_name": "安保部",
        "group_name": "安保巡逻组",
        "wechat_bound": true,
        "is_online": true,
        "status": "on_duty",
        "created_at": "2026-03-01 09:00:00"
      }
    ]
  }
}
```

### 9.2 创建/编辑用户

```
POST /api/v1/users
PUT  /api/v1/users/:id
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "name": "李四",
  "phone": "13800138001",
  "role": "handler",
  "department_id": 3,
  "group_ids": [10, 11]
}
```

### 9.3 用户组 CRUD

```
GET    /api/v1/user-groups
POST   /api/v1/user-groups
PUT    /api/v1/user-groups/:id
DELETE /api/v1/user-groups/:id
```

### 9.4 部门 CRUD

```
GET    /api/v1/departments
POST   /api/v1/departments
PUT    /api/v1/departments/:id
DELETE /api/v1/departments/:id
```

### 9.5 禁用/启用用户

```
DELETE /api/v1/users/:id
Authorization: Bearer <token>
```

**说明**：不会物理删除用户，只设置 `status = disabled`。管理员可以重新启用。

**请求体：**（无，仅需路径参数 `id`）

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1005,
    "name": "王五",
    "status": "disabled",
    "disabled_at": "2026-06-14 17:00:00"
  }
}
```

**错误场景：**
```json
{ "code": 40300, "message": "仅管理员可禁用用户", "data": null }
{ "code": 40002, "message": "不能禁用自己", "data": null }
{ "code": 40003, "message": "不能禁用超级管理员", "data": null }
{ "code": 40400, "message": "用户不存在", "data": null }
```

### 9.6 单独设置用户权限

为单个用户设置额外权限（覆盖角色默认权限），用于例外情况。

```
PUT /api/v1/users/:id/permissions
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "extra_permissions": ["schedule.edit", "schedule.import"],
  "revoke_permissions": ["work_order.delete"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `extra_permissions` | string[] | ☐ | 在角色默认权限之上**额外授予**的权限项 |
| `revoke_permissions` | string[] | ☐ | 在角色默认权限之上**额外收回**的权限项 |

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1003,
    "name": "李四",
    "role": "handler",
    "role_default_permissions": ["work_order.view", "work_order.accept", ...],
    "extra_permissions": ["schedule.edit", "schedule.import"],
    "revoke_permissions": ["work_order.delete"],
    "effective_permissions": ["work_order.view", "work_order.accept", "schedule.edit", "schedule.import"]
  }
}
```

权限项枚举：
```
work_order.view       work_order.accept      work_order.process
work_order.transfer   work_order.delete      work_order.export
template.view         template.create        template.edit
template.toggle
suppression.view      suppression.create     suppression.edit
suppression.delete
routing.view          routing.create         routing.edit         routing.delete
sla.view              sla.create             sla.edit             sla.delete
user.view             user.create            user.edit            user.disable
department.view       department.edit
group.view            group.edit
schedule.view         schedule.edit          schedule.import
permissions.view      permissions.edit
stats.view            stats.export
```

---

## 10. 排班管理

排班管理提供 7 个接口，覆盖查看、手动编辑、批量设置、Excel 导入、模板下载、换班申请全流程。

### 10.1 查看排班

```
GET /api/v1/schedules?date=2026-06-14&department_id=3
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `date` | string | ✅ | 查询日期，格式 YYYY-MM-DD |
| `department_id` | int | ☐ | 部门 ID，不传则按 JWT 角色自动限定范围 |

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "date": "2026-06-14",
    "department_name": "安保部",
    "shifts": [
      {
        "type": "day",
        "type_name": "白班",
        "time": "08:00-18:00",
        "users": [
          {"id": 1001, "name": "张三", "phone": "138****8000", "group_name": "安保巡逻组", "area": "B1停车场", "is_on_call": false},
          {"id": 1002, "name": "李四", "phone": "138****8001", "group_name": "安保巡逻组", "area": "B1停车场", "is_on_call": true}
        ]
      },
      {
        "type": "night",
        "type_name": "夜班",
        "time": "18:00-08:00",
        "users": [
          {"id": 1005, "name": "王五", "phone": "138****8002", "group_name": "安保巡逻组", "area": "B1停车场", "is_on_call": true},
          {"id": 1006, "name": "赵六", "phone": "138****8003", "group_name": "安保巡逻组", "area": "B1停车场", "is_on_call": false}
        ]
      }
    ]
  }
}
```

**错误场景：**
```json
{ "code": 40000, "message": "date 格式错误，需要 YYYY-MM-DD", "data": null }
```

---

### 10.2 手动设置排班（新增/覆盖某一天）

```
POST /api/v1/schedules
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "date": "2026-06-14",
  "department_id": 3,
  "shifts": [
    {
      "type": "day",
      "user_ids": [1001, 1002],
      "on_call_user_id": 1002
    },
    {
      "type": "night",
      "user_ids": [1005, 1006],
      "on_call_user_id": 1005
    }
  ]
}
```

**请求体字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `date` | string | ✅ | 排班日期 YYYY-MM-DD |
| `department_id` | int | ✅ | 部门 ID |
| `shifts[].type` | string | ✅ | 班次类型: `day`(白班) / `night`(夜班) |
| `shifts[].user_ids` | int[] | ✅ | 该班次人员 ID 列表 |
| `shifts[].on_call_user_id` | int | ☐ | 值班人 ID（优先接单） |

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 88,
    "date": "2026-06-14",
    "created_count": 2,
    "shifts": [
      {"type": "day", "user_count": 2, "on_call_user": "李四"},
      {"type": "night", "user_count": 2, "on_call_user": "王五"}
    ]
  }
}
```

**错误场景：**
```json
{ "code": 40001, "message": "用户 1005 当日已有排班（白班），不能重复排班", "data": null }
```

---

### 10.3 单独修改某条排班记录

```
PUT /api/v1/schedules/:id
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "shift_type": "night",
  "user_ids": [1005, 1006, 1008],
  "on_call_user_id": 1008,
  "area": "B2停车场"
}
```

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 88,
    "date": "2026-06-14",
    "updated_at": "2026-06-14 10:30:00"
  }
}
```

**错误场景：**
```json
{ "code": 40001, "message": "被修改用户 1005 当日已有夜班排班", "data": null }
{ "code": 40300, "message": "无权限修改此排班（仅经理及以上可修改）", "data": null }
```

---

### 10.4 批量排班

```
POST /api/v1/schedules/batch
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "start_date": "2026-06-15",
  "end_date": "2026-06-21",
  "department_id": 3,
  "pattern": {
    "day": {
      "user_ids": [1001, 1002],
      "on_call_user_id": 1002
    },
    "night": {
      "user_ids": [1005, 1006],
      "on_call_user_id": 1005
    }
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `start_date` | string | ✅ | 起始日期 YYYY-MM-DD |
| `end_date` | string | ✅ | 结束日期 YYYY-MM-DD |
| `department_id` | int | ✅ | 部门 ID |
| `pattern.day` | object | ✅ | 白班人员配置 |
| `pattern.night` | object | ✅ | 夜班人员配置 |

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_days": 7,
    "created_count": 14,
    "period": "2026-06-15 ~ 2026-06-21"
  }
}
```

**错误场景：**
```json
{ "code": 40000, "message": "start_date 不能晚于 end_date", "data": null }
{ "code": 40001, "message": "用户 1001 在 2026-06-17 已有排班冲突", "data": {"conflict_date": "2026-06-17", "user_id": 1001} }
```

---

### 10.5 Excel 排班导入

```
POST /api/v1/schedules/import
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**请求体（multipart/form-data）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `file` | file (.xlsx/.xls) | ✅ | 排班表 Excel 文件 |
| `department_id` | int | ✅ | 目标部门 ID |
| `mode` | string | ☐ | 冲突处理模式: `preview`(仅预览,默认) / `force`(强制覆盖) / `skip_conflict`(跳过冲突) |

**Excel 模板格式：**

| 日期 | 班次 | 姓名 | 手机号 | 负责区域 | 是否值班 |
|------|------|------|--------|----------|----------|
| 2026-06-14 | 白班 | 张三 | 13800138000 | B1停车场 | 否 |
| 2026-06-14 | 白班 | 李四 | 13800138001 | B1停车场 | 是 |
| 2026-06-14 | 夜班 | 王五 | 13800138002 | B1停车场 | 是 |

**响应（200，preview 模式）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "mode": "preview",
    "summary": {
      "total_rows": 15,
      "new_count": 12,
      "overwrite_count": 2,
      "conflict_count": 1
    },
    "rows": [
      {
        "row_number": 1,
        "date": "2026-06-14",
        "shift_type": "day",
        "name": "张三",
        "phone": "13800138000",
        "area": "B1停车场",
        "is_on_call": false,
        "status": "new",
        "status_label": "新增"
      },
      {
        "row_number": 2,
        "date": "2026-06-14",
        "shift_type": "day",
        "name": "李四",
        "phone": "13800138001",
        "area": "B1停车场",
        "is_on_call": true,
        "status": "new",
        "status_label": "新增"
      },
      {
        "row_number": 3,
        "date": "2026-06-14",
        "shift_type": "night",
        "name": "王五",
        "phone": "13800138002",
        "area": "B1停车场",
        "is_on_call": true,
        "status": "overwrite",
        "status_label": "覆盖原排班",
        "existing_schedule_id": 72
      },
      {
        "row_number": 4,
        "date": "2026-06-15",
        "shift_type": "day",
        "name": "张三",
        "phone": "13800138000",
        "area": "外围",
        "is_on_call": false,
        "status": "conflict",
        "status_label": "与已有排班冲突",
        "conflict_detail": "张三在 2026-06-14 已被排为白班(B1停车场)，同日不能排两个班"
      }
    ]
  }
}
```

**status 枚举：**

| status | 含义 | 后续操作 |
|--------|------|----------|
| `new` | 新记录，无冲突 | 可直接导入 |
| `overwrite` | 已有排班，将被覆盖 | 确认后覆盖 |
| `conflict` | 数据冲突 | 需人工处理或跳过 |
| `invalid` | 数据格式错误（如手机号未匹配到用户） | 需修正后重新导入 |

**响应（200，force 模式）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "mode": "force",
    "imported_count": 14,
    "skipped_count": 1,
    "skipped_rows": [4],
    "effect": "排班已生效，工单接收人已自动关联"
  }
}
```

**错误场景：**
```json
{ "code": 40000, "message": "文件格式不支持，请上传 .xlsx 或 .xls 文件", "data": null }
{ "code": 40001, "message": "Excel 表头不正确，缺少必填列: [日期, 班次, 姓名]", "data": null }
{ "code": 40401, "message": "以下手机号未匹配到用户: [13800000000, 13900000000]", "data": {"unmatched_phones": ["13800000000", "13900000000"]} }
```

---

### 10.6 下载导入模板

```
GET /api/v1/schedules/template
Authorization: Bearer <token>
```

**响应（200）：**
```
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename="WDOS_排班导入模板.xlsx"

[Excel 二进制文件流]
```

**模板内容：**

| 日期 | 班次 | 姓名 | 手机号 | 负责区域 | 是否值班 |
|------|------|------|--------|----------|----------|
| (示例) 2026-06-14 | 白班 | 张三 | 13800138000 | B1停车场 | 否 |

- 第一行为示例数据，用户替换为实际排班
- 班次列有下拉验证（白班/夜班）
- 是否值班列有下拉验证（是/否）

---

### 10.7 换班申请

```
POST /api/v1/schedules/swap
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "my_schedule_id": 88,
  "target_schedule_id": 95,
  "reason": "6月14日家中有事，申请与李四换班"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `my_schedule_id` | int | ✅ | 我的排班记录 ID |
| `target_schedule_id` | int | ✅ | 目标排班记录 ID（对方的） |
| `reason` | string | ✅ | 换班理由 |

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "swap_id": 12,
    "status": "pending_approval",
    "status_label": "等待班长审批",
    "approver": {"id": 1010, "name": "赵六", "role": "supervisor"},
    "detail": {
      "applicant": {"id": 1001, "name": "张三", "date": "2026-06-14", "shift": "白班"},
      "target": {"id": 1002, "name": "李四", "date": "2026-06-16", "shift": "白班"}
    }
  }
}
```

**换班状态流转：**

```
pending_approval（等待审批）
    ├── 班长审批通过 → approved → 排班自动互换
    └── 班长驳回 → rejected → 排班不变
```

**审批接口（班长/经理调用）：**
```
POST /api/v1/schedules/swap/:swap_id/approve
Authorization: Bearer <token>

请求体:
{ "action": "approve", "comment": "同意换班" }
// 或
{ "action": "reject", "comment": "该时段人员不足，无法换班" }
```

**响应（审批通过）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "swap_id": 12,
    "status": "approved",
    "effect": "排班已互换: 张三 6/14白班 ↔ 李四 6/16白班"
  }
}
```

**错误场景：**
```json
{ "code": 40000, "message": "不能和自己换班", "data": null }
{ "code": 40001, "message": "目标排班日期(6/16)与你的排班日期(6/14)不同，无法互换", "data": null }
{ "code": 40002, "message": "目标排班已被其他人申请换班，请等待当前申请处理完毕", "data": null }
{ "code": 40300, "message": "无权审批此换班申请（仅班长/经理可审批）", "data": null }
```

---

## 11. 权限管理

### 11.1 获取角色权限配置

```
GET /api/v1/permissions/roles?role=supervisor
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `role` | string | ☐ | 角色名。不传则返回所有角色 |

**响应（200，指定角色）：**
```json
{
  "code": 0,
  "data": {
    "role": "supervisor",
    "role_name": "领班",
    "permissions": [
      {
        "module": "工单中心",
        "actions": {
          "view": true,
          "edit": true,
          "delete": false
        },
        "scope": "group"
      },
      {
        "module": "工单数据",
        "actions": {
          "view": true,
          "transfer": true,
          "delete": false,
          "export": true
        },
        "scope": "group"
      },
      {
        "module": "排班管理",
        "actions": {
          "view": true,
          "edit": true,
          "import": true,
          "modify": false
        },
        "scope": "group",
        "note": "排班修改需经理级别"
      },
      {
        "module": "统计报表",
        "actions": {
          "view": true,
          "export": true
        },
        "scope": "group"
      }
    ]
  }
}
```

| 字段 | 说明 |
|------|------|
| `scope` | `self`(个人) / `group`(组内) / `department`(部门) / `global`(全局) |

### 11.2 保存角色权限配置

保存后立即生效，该角色下所有用户的权限实时更新。

```
PUT /api/v1/permissions/roles/:role
Authorization: Bearer <token>
```

**请求体：**
```json
{
  "permissions": [
    {
      "module": "工单中心",
      "actions": { "view": true, "edit": true, "delete": false },
      "scope": "group"
    },
    {
      "module": "排班管理",
      "actions": { "view": true, "edit": false, "import": false, "modify": false },
      "scope": "group"
    },
    {
      "module": "统计报表",
      "actions": { "view": true, "export": false },
      "scope": "group"
    }
  ]
}
```

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "role": "supervisor",
    "updated_modules": 3,
    "affected_users": 8,
    "updated_at": "2026-06-14 17:30:00"
  }
}
```

**错误场景：**
```json
{ "code": 40300, "message": "仅管理员可配置权限", "data": null }
{ "code": 40000, "message": "权限模块不存在: [unknown_module]", "data": null }
```

---

## 12. 统计接口

**角色联动规则**：所有统计接口根据 JWT token 中的角色自动限定数据范围，同一 URL 不同角色看到不同数据。小程序和管理后台共用同一接口，前端不需要传范围参数。

| 角色 | 数据范围 |
|------|----------|
| handler | 个人数据 |
| supervisor | 所在组数据 |
| manager | 所在部门数据 |
| director/admin | 全商场数据 |

---

### 12.1 个人统计概览

小程序「我的」页面使用。

```
GET /api/v1/stats/my-overview
Authorization: Bearer <token>
```

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "today": {
      "pending_count": 5,
      "processing_count": 2,
      "completed_count": 12,
      "avg_accept_seconds": 18,
      "avg_process_seconds": 95,
      "overtime_count": 1
    },
    "week": {
      "completed_count": 68,
      "avg_accept_seconds": 22,
      "avg_process_seconds": 110,
      "overtime_count": 3,
      "rank": 2,
      "total_in_group": 8
    }
  }
}
```

---

### 12.2 每日报警概览

小程序统计 Tab 顶部卡片 + 管理后台首页使用。

```
GET /api/v1/stats/daily-overview?date=2026-06-14
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `date` | string | ✅ | 查询日期 YYYY-MM-DD |

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "date": "2026-06-14",
    "total_alarms": 1258,
    "prev_day_total": 1123,
    "change_rate": 0.12,
    "change_direction": "up",
    "total_orders": 1185,
    "completed_orders": 1116,
    "completion_rate": 0.942,
    "overtime_orders": 47,
    "overtime_rate": 0.04,
    "avg_accept_seconds": 25,
    "avg_process_seconds": 132,
    "suppressed_alarms": 10424,
    "suppress_rate": 0.892
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `change_rate` | float | 环比变化率（正=上升，负=下降） |
| `change_direction` | string | `up` / `down` / `flat` |
| `suppress_rate` | float | 被抑制的报警 / 总报警 |

---

### 12.3 每类算法报警数

小程序统计 Tab 横向柱状图 + 管理后台统计页使用。

```
GET /api/v1/stats/by-algorithm?date=2026-06-14
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `date` | string | ✅ | 查询日期 YYYY-MM-DD |

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "date": "2026-06-14",
    "total": 1258,
    "items": [
      {
        "algorithm_id": 4,
        "algorithm_name": "行人闯入",
        "algorithm_name_en": "CR_PERSON_INVASION",
        "count": 523,
        "ratio": 0.416,
        "order_count": 98,
        "completion_rate": 0.93,
        "avg_process_seconds": 48
      },
      {
        "algorithm_id": 1,
        "algorithm_name": "烟雾检测",
        "algorithm_name_en": "CR_SMOKE_DETECTION",
        "count": 318,
        "ratio": 0.253,
        "order_count": 285,
        "completion_rate": 0.96,
        "avg_process_seconds": 55
      },
      {
        "algorithm_id": 6,
        "algorithm_name": "人员聚集",
        "algorithm_name_en": "CR_CROWD_GATHERING",
        "count": 215,
        "ratio": 0.171,
        "order_count": 190,
        "completion_rate": 0.94,
        "avg_process_seconds": 62
      }
    ]
  }
}
```

---

### 12.4 各大区域统计

小程序统计 Tab 区域表 + 管理后台统计页使用。

```
GET /api/v1/stats/by-area?date=2026-06-14
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `date` | string | ✅ | 查询日期 YYYY-MM-DD |

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "date": "2026-06-14",
    "total_alarms": 1258,
    "areas": [
      {
        "area_name": "B1停车场",
        "camera_group_pattern": "parking_B1",
        "alarm_count": 412,
        "order_count": 380,
        "completed_count": 354,
        "completion_rate": 0.932,
        "avg_process_seconds": 48,
        "overtime_count": 5,
        "handler_group_name": "安保巡逻组"
      },
      {
        "area_name": "1楼 (F1)",
        "camera_group_pattern": "public_F1",
        "alarm_count": 285,
        "order_count": 270,
        "completed_count": 256,
        "completion_rate": 0.947,
        "avg_process_seconds": 45,
        "overtime_count": 3,
        "handler_group_name": "商管巡逻组"
      },
      {
        "area_name": "2楼 (F2)",
        "camera_group_pattern": "equipment_F2",
        "alarm_count": 156,
        "order_count": 148,
        "completed_count": 142,
        "completion_rate": 0.962,
        "avg_process_seconds": 55,
        "overtime_count": 1,
        "handler_group_name": "工程值班组"
      }
    ]
  }
}
```

---

### 12.5 处理耗时分布

小程序统计 Tab 耗时分布图 + 管理后台统计页使用。

```
GET /api/v1/stats/process-time-distribution?date=2026-06-14
Authorization: Bearer <token>
```

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "date": "2026-06-14",
    "total_completed": 1116,
    "buckets": [
      {"label": "0-30秒",  "min_seconds": 0,   "max_seconds": 30,  "count": 502, "ratio": 0.45},
      {"label": "30-60秒", "min_seconds": 30,  "max_seconds": 60,  "count": 313, "ratio": 0.28},
      {"label": "60-120秒","min_seconds": 60,  "max_seconds": 120, "count": 179, "ratio": 0.16},
      {"label": "120-300秒","min_seconds": 120, "max_seconds": 300, "count": 89,  "ratio": 0.08},
      {"label": ">300秒",  "min_seconds": 300, "max_seconds": null,"count": 33,  "ratio": 0.03}
    ]
  }
}
```

---

### 12.6 报警趋势（近N天）

管理后台统计页折线图使用。

```
GET /api/v1/stats/trend?days=7
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `days` | int | ✅ | 查询天数，最大 90 |

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "period": "2026-06-08 ~ 2026-06-14",
    "days": 7,
    "points": [
      {
        "date": "2026-06-08",
        "alarm_count": 890,
        "order_count": 850,
        "completed_count": 810,
        "overtime_count": 15,
        "completion_rate": 0.953,
        "avg_process_seconds": 55
      },
      {
        "date": "2026-06-09",
        "alarm_count": 920,
        "order_count": 880,
        "completed_count": 845,
        "overtime_count": 12,
        "completion_rate": 0.96,
        "avg_process_seconds": 52
      },
      {
        "date": "2026-06-14",
        "alarm_count": 1258,
        "order_count": 1185,
        "completed_count": 1116,
        "overtime_count": 47,
        "completion_rate": 0.942,
        "avg_process_seconds": 48
      }
    ]
  }
}
```

**points 按日期升序排列**，第一个元素是最早一天。

---

### 12.7 人员绩效排行

管理后台统计页排行榜使用。

```
GET /api/v1/stats/user-ranking?date=2026-06-14&sort_by=completed_count&order=desc&page=1&size=20
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `date` | string | ✅ | 查询日期 YYYY-MM-DD |
| `sort_by` | string | ☐ | 排序字段：`completed_count`(默认) / `avg_process_seconds` / `overtime_count` |
| `order` | string | ☐ | `asc` / `desc`(默认) |
| `page` | int | ☐ | 页码，默认 1 |
| `size` | int | ☐ | 每页条数，默认 20，最大 100 |

**角色限定**：
- handler 只能看到自己（返回单条或空）
- supervisor 看到本组所有人
- manager 看到本部门所有人
- director/admin 看到全商场所有人

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "date": "2026-06-14",
    "total": 85,
    "page": 1,
    "size": 20,
    "sort_by": "completed_count",
    "order": "desc",
    "list": [
      {
        "rank": 1,
        "user_id": 1001,
        "user_name": "张三",
        "department_name": "安保部",
        "group_name": "安保巡逻组",
        "completed_count": 12,
        "avg_accept_seconds": 18,
        "avg_process_seconds": 27,
        "total_avg_seconds": 45,
        "overtime_count": 2,
        "on_duty": true
      },
      {
        "rank": 2,
        "user_id": 1002,
        "user_name": "李四",
        "department_name": "安保部",
        "group_name": "安保巡逻组",
        "completed_count": 10,
        "avg_accept_seconds": 22,
        "avg_process_seconds": 30,
        "total_avg_seconds": 52,
        "overtime_count": 1,
        "on_duty": true
      }
    ]
  }
}
```

---

## 13. WebSocket 通知

### 13.1 连接

```
GET wss://wdos.yourmall.com/ws/notifications?token=<jwt_token>
```

### 13.2 推送消息格式

```json
{
  "type": "new_order",
  "data": {
    "order_id": 1525,
    "title": "行人闯入 - B1停车场C区3号通道",
    "camera_name": "B1停车场C区3号通道",
    "algorithm_name": "行人闯入",
    "degree": 3,
    "created_at": "2026-06-14 15:04:05"
  }
}
```

**消息类型枚举：**

| type | 说明 | 触发条件 |
|------|------|----------|
| `new_order` | 新工单 | 新报警生成工单 |
| `order_accepted` | 工单被接 | 有人接单 |
| `order_transferred` | 工单转交 | 转交给你或从你转出 |
| `escalation_l1` | 一级超时 | 接单/处理超时触发 30s/150s |
| `escalation_l2` | 二级超时 | 接单/处理超时触发 120s/300s |
| `escalation_l3` | 三级超时 | 接单/处理超时触发 300s/600s |
| `order_completed` | 工单完成 | 处理完成 |
| `lock_applied` | 锁定生效 | 点位被锁定 |
| `lock_released` | 锁定解除 | 点位解锁 |
| `summary_report` | 抑制摘要 | 解锁后推送抑制期间报警摘要 |

---

## 14. 通知历史

### 14.1 获取通知列表

小程序「消息」Tab 使用。

```
GET /api/v1/notifications?page=1&size=20
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `page` | int | ☐ | 页码，默认 1 |
| `size` | int | ☐ | 每页条数，默认 20 |
| `unread_only` | bool | ☐ | 仅未读，默认 false |

**响应（200）：**
```json
{
  "code": 0,
  "data": {
    "total": 128,
    "unread_count": 3,
    "page": 1,
    "size": 20,
    "list": [
      {
        "id": 15601,
        "type": "new_order",
        "title": "新工单",
        "message": "行人闯入 - B1停车场C区3号通道",
        "order_id": 1525,
        "order_no": "WD-20260614-01525",
        "is_read": false,
        "created_at": "2026-06-14 15:04:05"
      },
      {
        "id": 15600,
        "type": "escalation_l1",
        "title": "接单超时提醒",
        "message": "WD-1525 已超时30秒未接单，已上报班长",
        "order_id": 1525,
        "is_read": true,
        "created_at": "2026-06-14 15:04:35"
      }
    ]
  }
}
```

**type 枚举：** 见 13.2 节 WebSocket 推送消息类型（通知列表与原 WebSocket 推送使用同一套 type）

### 14.2 标记已读

```
POST /api/v1/notifications/read
Authorization: Bearer <token>
```

**请求体（标记单条）：**
```json
{
  "notification_ids": [15601, 15600]
}
```

**请求体（全部已读）：**
```json
{
  "mark_all": true
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|:--:|------|
| `notification_ids` | int[] | ☐ | 要标记的通知 ID 列表 |
| `mark_all` | bool | ☐ | 全部标记已读（传 true 时忽略 notification_ids） |

**响应（200）：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "marked_count": 2,
    "remaining_unread": 1
  }
}
```

---

## 15. 通用规范

### 15.1 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 15.2 错误码

| code | 说明 |
|------|------|
| 0 | 成功 |
| 40000 | 参数错误 |
| 40001 | 工单已被他人接单 |
| 40002 | 工单状态不允许此操作 |
| 40003 | 无权限操作此工单 |
| 40004 | 转交目标用户无接单权限 |
| 40100 | 未登录或 token 过期 |
| 40300 | 无权限 |
| 40400 | 资源不存在 |
| 50000 | 服务器内部错误 |

### 15.3 分页规范

```
请求参数:
  page    int  页码（从1开始）
  size    int  每页条数（默认20，最大100）

响应字段:
  total   int  总记录数
  page    int  当前页码
  size    int  每页条数
  list    array 数据列表
```

### 15.4 日期时间格式

统一使用：`YYYY-MM-DD HH:mm:ss`（如 `2026-06-14 15:04:05`）

---

> **文档状态**：v1.0，覆盖 MVP 全部接口 + 后续迭代接口
> **下一步**：生成 Swagger JSON/YAML → 导入 swaggo 自动生成文档页面
