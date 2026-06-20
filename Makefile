.PHONY: run build test lint dev db-up db-down docs

# 开发运行
run:
	go run ./cmd/api

# 编译
build:
	go build -o bin/wdos-api ./cmd/api

# 测试
test:
	go test ./... -v -cover

# 代码检查
lint:
	golangci-lint run ./...

# 生成 Swagger 文档
docs:
	swag init -g cmd/api/main.go -o docs

# 启动基础设施
db-up:
	docker-compose -f deploy/docker-compose.yml up -d

# 停止基础设施
db-down:
	docker-compose -f deploy/docker-compose.yml down

# 数据库迁移
migrate:
	go run ./cmd/migrate/main.go

# 开发环境一键启动
dev: db-up run
