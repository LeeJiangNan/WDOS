# WDOS - 商场 AI 工单调度与编排系统

基于鲲云 CRIP 人工智能推理平台的商场工单系统，将 AI 视频分析报警自动转化为可追踪的工单流程。

## 技术栈

| 组件 | 选型 |
|------|------|
| 后端 | Go 1.22+ / Gin / GORM |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7 |
| 对象存储 | MinIO |
| 管理后台 | Vue 3 + Element Plus |
| 移动端 | 微信小程序 (Taro + React) |

## 快速开始

### 1. 启动基础设施

```bash
make db-up
```

### 2. 运行 API 服务

```bash
cp config/config.yaml config/config.local.yaml
# 编辑 config.local.yaml，填入实际配置

make run
```

### 3. 访问

- API: http://localhost:8080
- Swagger: http://localhost:8080/swagger/index.html
- MinIO 控制台: http://localhost:9001 (minioadmin/minioadmin123)

## 项目结构

```
cmd/          入口
internal/     核心业务代码
  transport/  接入层（HTTP路由、Callback接收）
  service/    业务逻辑层
  repository/ 数据访问层
  model/      数据模型
config/       配置文件
deploy/       Docker Compose
docs/         Swagger 文档
migrations/   数据库迁移脚本
web/          Vue 3 管理后台
miniapp/      微信小程序
```

## 设计文档

- [架构设计方案](docs/design/WDOS%E5%95%86%E5%9C%BAAI%E5%B7%A5%E5%8D%95%E7%B3%BB%E7%BB%9F%E6%9E%B6%E6%9E%84%E8%AE%BE%E8%AE%A1%E6%96%B9%E6%A1%88v2.0.md)
- [API 接口定义](docs/WDOS_API%E6%8E%A5%E5%8F%A3%E5%AE%9A%E4%B9%89%E6%96%87%E6%A1%A3.md)
- [业务流程闭环图](docs/WDOS_%E4%B8%9A%E5%8A%A1%E6%B5%81%E7%A8%8B%E9%97%AD%E7%8E%AF%E5%9B%BE.md)
- [权限矩阵总表](docs/WDOS_%E6%9D%83%E9%99%90%E7%9F%A9%E9%98%B5%E6%80%BB%E8%A1%A8.md)
- [接口对账表](docs/WDOS_%E6%8E%A5%E5%8F%A3%E5%AF%B9%E8%B4%A6%E8%A1%A8.md)

## 开发进度

| 阶段 | 状态 | 内容 |
|------|:--:|------|
| 1. 基础设施 | ✅ | MySQL/Redis/MinIO 连接 + 12 张表 |
| 2.1 Callback 接收器 | ✅ | 去重/图片下载/存储/抑制/工单生成 |
| 2.2 手动补偿接口 | ✅ | 调 CRIP OpenAPI 分页拉取补漏 |
| 2.3 认证模块          | ✅ | `073936f`
| 2.4-2.11 其余 P0       | ⏳ | 下一步 | 待开发 |
| 3. P0 前端 | ⏳ | 管理后台 + 小程序 |
| 4. P1 增强 | ⏳ | 锁定/路由/看板/统计 |
| 5-6. 测试上线 | ⏳ | 待进行 |

## 分支策略

- `main` — 稳定版本
- `develop` — 开发主线
- `feature/*` — 功能分支
- `hotfix/*` — 紧急修复
