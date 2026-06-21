# WDOS 开发规范

> 生成时间：2026-06-21
> 来源：扫描 `github.com/LeeJiangNan/WDOS` 代码仓

## 规范文件

| 文件 | 覆盖范围 |
|------|----------|
| [go-project-structure.md](go-project-structure.md) | 目录布局、包命名、依赖方向 |
| [models.md](models.md) | GORM 模型定义规范、JSON列处理、表名约定 |
| [services.md](services.md) | Service 层模式、构造函数、错误处理 |
| [api-responses.md](api-responses.md) | 统一 API 响应格式、错误码 |
| [config-and-infra.md](config-and-infra.md) | 配置加载、日志、数据库/缓存/存储连接 |

## 开发踩坑记录

| 日期 | 问题 | 解决方案 | 相关文件 |
|------|------|----------|----------|
| 06-21 | MySQL JSON 列不接受空字符串 | 使用 `*string` 指针类型或设默认值 `"{}"` | `internal/model/work_order.go:25` |
| 06-21 | macOS dyld 报 `missing LC_UUID` | 编译加 `CGO_ENABLED=0` | `Makefile` |
| 06-21 | Redis 去重 key 不设前缀会冲突 | 全局使用 `wdos:` 前缀，从 config 读取 | `config/config.yaml:24` |
