# GORM 模型规范

## 基础结构

每个模型文件包含：
1. struct 定义 + gorm 标签
2. `TableName()` 方法
3. JSON 标签（与 gorm 列名一致，统一 snake_case）

```go
type WorkOrder struct {
    ID     uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
    Status string `gorm:"type:enum('pending','processing','completed');default:pending;index" json:"status"`
}

func (WorkOrder) TableName() string { return "work_order" }
```

参照文件：`internal/model/work_order.go`

## gorm 标签规范

| 场景 | 标签写法 |
|------|----------|
| 主键 | `gorm:"primaryKey;autoIncrement"` |
| 唯一索引 | `gorm:"uniqueIndex;size:64;not null"` |
| 普通索引 | `gorm:"index:idx_status"` |
| 联合索引 | `gorm:"index:idx_camera_algo"` 两字段用同一个 index 名 |
| 默认值 | `gorm:"default:false"` 或 `gorm:"default:'callback'"` |
| 枚举 | `gorm:"type:enum('a','b','c');default:a"` |
| JSON 列 | `gorm:"type:json"` |
| 注释 | `gorm:"comment:字段说明"` |
| 自动时间 | `gorm:"autoCreateTime"` / `gorm:"autoUpdateTime"` |

## JSON 列处理 ⚠️ 重要

MySQL JSON 列**不接受空字符串**，必须处理为 NULL 或有效 JSON：

```go
// ✅ 正确：使用指针类型，空值写 NULL
FormData *string `gorm:"type:json" json:"form_data"`

// ✅ 正确：Service 层设默认值
defaultFormData := "{}"
order.FormData = &defaultFormData

// ❌ 禁止：string 类型 + 空字符串
FormData string `gorm:"type:json"`  // 存入空串 → MySQL 3140 错误
```

踩坑记录：2026-06-21，`work_order.form_data` 空字符串导致 `Error 3140: Invalid JSON text`。

参照文件：`internal/model/work_order.go:25`、`internal/service/alarm/service.go:138`

## 时间字段

- 业务时间用 `time.Time`
- 可为空的时间用 `*time.Time`（指针类型，存 NULL）
- gorm 自动管理 `autoCreateTime` / `autoUpdateTime`

```go
SlaAcceptDeadline *time.Time  // 可为空
CreatedAt         time.Time   `gorm:"autoCreateTime"`  // 必填
```

## 表名

全部 snake_case，不加前缀：

```
crip_alarm_raw, work_order, work_order_log,
work_order_template, suppression_rule, area_routing_rule,
sla_escalation_policy, staff_schedule, workflow_definition,
users, departments, user_groups
```

参照文件：`internal/model/*.go` 共 12 个文件
