// Package main WDOS API 服务入口
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	logger := initLogger(cfg)

	// 初始化数据库
	db := initDatabase(cfg)
	rdb := initRedis(cfg)
	minioClient := initMinio(cfg)

	// 初始化仓储层
	repos := initRepositories(db, rdb, minioClient)

	// 初始化服务层
	svcs := initServices(repos, cfg, logger)

	// 初始化路由
	engine := gin.Default()
	registerMiddleware(engine, svcs.Auth)
	registerRoutes(engine, svcs)

	// 启动 HTTP 服务
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: engine,
	}

	go func() {
		logger.Infof("WDOS API 服务启动, 端口: %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	logger.Info("服务已关闭")
}
