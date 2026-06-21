# 配置与基础设施规范

## 配置文件

单一配置文件 `config/config.yaml`，使用 viper 加载。所有环境差异通过 YAML 和环境变量注入。

```yaml
database:
  host: "127.0.0.1"
  port: 3307
  password: "wdos123"   # 开发环境直接写，生产用 ${WDOS_DB_PASSWORD}
```

参照文件：`config/config.yaml`、`internal/pkg/config/config.go`

## 配置结构体

Config struct 必须加 `mapstructure` 标签，子结构体独立定义：

```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    SLA      SLAConfig      `mapstructure:"sla"`
}

type DatabaseConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
    Name string `mapstructure:"name"`
}
```

DSN 方法放子结构体上：`func (d DatabaseConfig) DSN() string`

参照文件：`internal/pkg/config/config.go:10-87`

## 日志

使用 `go.uber.org/zap`，通过 `logger.New(mode)` 初始化：

- `debug` 模式：彩色控制台输出
- `release` 模式：JSON 格式输出

```go
sugar := logger.New(cfg.Server.Mode)
defer sugar.Sync()
```

参照文件：`internal/pkg/logger/logger.go`

## 数据库连接

GORM + MySQL，连接池参数固定：

```go
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

参照文件：`internal/repository/mysql/mysql.go:26-28`

创建新仓储时**不用再调这些参数**，复用 `Connect()` 函数即可。

## Redis 连接

go-redis v9，key 统一加前缀（从 config 读取）：

```go
prefix := cfg.Redis.Prefix  // "wdos:"
dedupKey := prefix + "alarm:" + snowflakeID
```

- 去重 key 带 24h TTL：`rdb.SetNX(ctx, key, "1", 24*time.Hour)`
- 心跳计数器：`rdb.Incr(ctx, prefix+"callback:count")`

参照文件：`internal/repository/redis/redis.go`、`internal/service/alarm/service.go:48-56`

## MinIO

minio-go v7，初始化时自动创建 bucket：

```go
exists, _ := client.BucketExists(ctx, bucket)
if !exists {
    client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}
```

图片存储路径：`alarms/raw/{snowflake_id}.jpg`

参照文件：`internal/repository/minio/minio.go:25-33`

## Docker Compose

基础设施用 Docker Compose 管理，端口映射到本地：

| 服务 | 容器端口 | 宿主机端口 |
|------|:--:|:--:|
| MySQL | 3306 | 3307 |
| Redis | 6379 | 6380 |
| MinIO API | 9000 | 9000 |
| MinIO Console | 9001 | 9001 |

参照文件：`deploy/docker-compose.yml`

## 编译注意

macOS 编译必须加 `CGO_ENABLED=0`，否则 dyld 报 `missing LC_UUID`：

```bash
CGO_ENABLED=0 go build -o /tmp/wdos-server ./cmd/api
```

踩坑记录：2026-06-21，不加此参数导致二进制无法在 macOS 运行。
