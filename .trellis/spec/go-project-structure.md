# Go 项目结构与包规范

## 目录布局

```
cmd/api/              # 唯一入口，只做依赖注入和启动
internal/
  transport/          # 接入层（HTTP 路由、Callback 接收）
  service/            # 业务逻辑层
  repository/         # 数据访问层（MySQL、Redis、MinIO）
  model/              # 数据模型（GORM struct）
  pkg/                # 内部工具包（config、logger）
pkg/                  # 外部可用工具包（response）
```

参照文件：`cmd/api/main.go`、`internal/service/alarm/service.go`

## 依赖方向

严格单向：`transport → service → repository/model`

```
transport  ← 可以 import service、model、pkg
service    ← 可以 import repository、model、pkg
repository ← 可以 import model、pkg
model      ← 不 import 任何 internal 包
```

## 包命名

- 包名 = 目录名，全小写，单数形式
- 避免 `util`、`common`、`base` 等无意义名称
- Redis 包因与标准库冲突，包名用 `redisx`，import 别名 `redisx "path"`

参照文件：`internal/repository/redis/redis.go:1`

## 构造函数模式

每个 service/repository 包导出单一构造函数 `New()`，返回具体类型指针：

```go
// ✅ 正确
func New(db *gorm.DB, rdb *redis.Client, ...) *Service {
    return &Service{db: db, rdb: rdb, ...}
}

// ❌ 禁止: 返回 interface、使用全局变量、init() 中初始化
```

参照文件：`internal/service/alarm/service.go:33-43`

## 入口文件

`cmd/api/main.go` 负责：
1. 加载配置
2. 初始化日志
3. 连接 MySQL/Redis/MinIO
4. 自动建表（AutoMigrate）
5. 初始化 service
6. 注册路由
7. 启动 HTTP + 优雅退出

不允许在 `main()` 里写业务逻辑。每步一个函数调用。
