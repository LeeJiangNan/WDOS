# Service 层规范

## 结构定义

每个 service 一个 struct，字段私有，构造函数 `New()` 注入依赖：

```go
type Service struct {
    db    *gorm.DB
    rdb   *redis.Client
    minio *minio.Client
    sugar *zap.SugaredLogger
}

func New(db *gorm.DB, rdb *redis.Client, ...) *Service {
    return &Service{db: db, rdb: rdb, ...}
}
```

参照文件：`internal/service/alarm/service.go:19-43`

## 方法签名

```go
// 业务方法：ctx 第一个参数，error 最后一个返回值
func (s *Service) ProcessCallback(ctx context.Context, cb *model.CRIPCallback) (*model.CallbackResponse, error)

// 内部辅助方法：小写开头，不导出
func (s *Service) downloadImage(ctx context.Context, url, objectName string) (string, error)
```

## 错误处理

Service 层只返回业务错误，不处理 HTTP 状态码：

```go
// ✅ Service 返回 error
if err := s.db.Create(raw).Error; err != nil {
    s.sugar.Errorf("存储原始报警失败: %v", err)
    return nil, fmt.Errorf("存储原始报警失败: %w", err)
}

// ❌ 禁止在 service 层调用 c.JSON()
```

HTTP 状态码和响应格式由 transport 层（`cmd/api/main.go` 的路由 handler）统一处理。

参照文件：`internal/service/alarm/service.go:130-134`

## 日志规范

使用注入的 `*zap.SugaredLogger`：

```go
s.sugar.Infof("工单已生成: snowflake=%s, order=%s", id, orderNo)
s.sugar.Warnf("下载图片失败: %v", err)
s.sugar.Errorf("存储失败: %v", err)
```

- Info：正常业务流程节点
- Warn：非致命异常（图片下载失败但不影响主流程）
- Error：需要人工关注的问题

## 事务处理

涉及多表写入用 GORM 事务：

```go
// 当前代码中 callback 处理是单表顺序写入，后续阶段如果
// crip_alarm_raw + work_order + work_order_log 必须原子写入，
// 应改为：
s.db.Transaction(func(tx *gorm.DB) error {
    tx.Create(raw)
    tx.Create(order)
    tx.Create(log)
    return nil
})
```

## 抑制检查模式

查询是否存在 + 条件判断的组合：

```go
var pendingOrder model.WorkOrder
err := s.db.Where("camera_name = ? AND algorithm_name = ? AND status IN ('pending','processing')",
    cb.CameraName, cb.AlgorithmName).First(&pendingOrder).Error
if err == nil {
    // 存在，抑制
}
```

参照文件：`internal/service/alarm/service.go:113-121`
