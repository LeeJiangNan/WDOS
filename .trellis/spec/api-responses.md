# API 响应规范

## 统一响应格式

所有 API 返回相同结构：

```json
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

参照文件：`pkg/response/response.go`

## 使用方式

```go
import "github.com/LeeJiangNan/WDOS/pkg/response"

// 成功
response.Success(c, data)

// 成功 + 自定义消息
response.SuccessMsg(c, "导入完成", data)

// 各类错误
response.BadRequest(c, "缺少 snowflake_id")
response.Unauthorized(c, "token 已过期")
response.Forbidden(c, "仅管理员可操作")
response.NotFound(c, "工单不存在")
response.ServerError(c, "处理报警失败")
```

## 错误码表

| 错误码 | 常量 | 含义 |
|:--:|------|------|
| 0 | `CodeSuccess` | 成功 |
| 40000 | `CodeBadRequest` | 参数错误 |
| 40001 | `CodeOrderTaken` | 工单已被他人接单 |
| 40002 | `CodeInvalidStatus` | 工单状态不允许此操作 |
| 40003 | `CodeNoPermission` | 无权限操作此工单 |
| 40004 | `CodeInvalidTarget` | 转交目标用户无接单权限 |
| 40100 | `CodeUnauthorized` | 未登录或 token 过期 |
| 40101 | `CodeTokenExpired` | token 已过期无法刷新 |
| 40300 | `CodeForbidden` | 无权限 |
| 40400 | `CodeNotFound` | 资源不存在 |
| 40401 | `CodeUserNotFound` | 用户不存在 |
| 50000 | `CodeServerError` | 服务器内部错误 |

## Callback 响应特殊格式

Callback 接口返回专用的 `CallbackResponse` 结构：

```json
{
    "code": 0,
    "data": {
        "action": "created",
        "work_order_id": 1,
        "suppressed": false,
        "reason": "成功生成工单"
    }
}
```

`action` 枚举：`created` / `suppressed` / `locked` / `ignored`

参照文件：`internal/model/callback.go:59-64`

## 不使用 gin.H 作为业务响应

```go
// ❌ 禁止在 handler 里拼 gin.H
c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{...}})

// ✅ 正确
response.Success(c, result)
```
