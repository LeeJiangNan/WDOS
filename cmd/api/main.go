// Package main WDOS API 服务入口
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/LeeJiangNan/WDOS/internal/model"
	"github.com/LeeJiangNan/WDOS/internal/pkg/config"
	"github.com/LeeJiangNan/WDOS/internal/pkg/logger"
	"github.com/LeeJiangNan/WDOS/internal/service/alarm"
	miniox "github.com/LeeJiangNan/WDOS/internal/repository/minio"
	"github.com/LeeJiangNan/WDOS/internal/repository/mysql"
	redisx "github.com/LeeJiangNan/WDOS/internal/repository/redis"
	"github.com/LeeJiangNan/WDOS/pkg/response"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfgPath := "config/config.yaml"
	if v := os.Getenv("WDOS_CONFIG"); v != "" {
		cfgPath = v
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化日志
	sugar := logger.New(cfg.Server.Mode)
	defer sugar.Sync()
	sugar.Infof("WDOS 启动中... 环境: %s", cfg.Server.Mode)

	// 3. 连接 MySQL
	db, err := mysql.Connect(cfg.Database.DSN(), cfg.Server.Mode == "debug")
	if err != nil {
		sugar.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 4. 自动建表（12 张表）
	if err := autoMigrate(db); err != nil {
		sugar.Fatalf("自动建表失败: %v", err)
	}
	sugar.Info("数据库表已同步 (12 张表)")

	// 5. 连接 Redis
	rdb, err := redisx.Connect(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		sugar.Fatalf("连接 Redis 失败: %v", err)
	}

	// 6. 连接 MinIO
	minioClient, err := miniox.Connect(
		cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey,
		cfg.MinIO.Bucket, cfg.MinIO.UseSSL,
	)
	if err != nil {
		sugar.Fatalf("连接 MinIO 失败: %v", err)
	}

	// 7. 初始化 Gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery(), gin.Logger())

	// 8. 初始化服务
	alarmSvc := alarm.New(db, rdb, minioClient, cfg.MinIO.Bucket, cfg.Redis.Prefix, cfg.CRIP, sugar)

	// 9. 注册路由
	registerRoutes(engine, alarmSvc, cfg, sugar)

	// 9. 启动 HTTP 服务
	addr := ":" + cfg.Server.Port
	srv := &http.Server{Addr: addr, Handler: engine}

	go func() {
		sugar.Infof("✅ WDOS API 服务已启动: http://localhost%s", addr)
		sugar.Infof("   Swagger 文档: http://localhost%s/swagger/index.html", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("服务异常退出: %v", err)
		}
	}()

	// 10. 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sugar.Info("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		sugar.Errorf("服务关闭异常: %v", err)
	}
	sugar.Info("服务已关闭")
}

// autoMigrate 自动建表
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.CRIPAlarmRaw{},
		&model.WorkOrder{},
		&model.WorkOrderLog{},
		&model.WorkOrderTemplate{},
		&model.SuppressionRule{},
		&model.AreaRoutingRule{},
		&model.SlaEscalationPolicy{},
		&model.StaffSchedule{},
		&model.WorkflowDefinition{},
		&model.User{},
		&model.Department{},
		&model.UserGroup{},
	)
}

// registerRoutes 注册所有路由
func registerRoutes(
	engine *gin.Engine,
	alarmSvc *alarm.Service,
	cfg *config.Config,
	sugar *zap.SugaredLogger,
) {
	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":  "ok",
			"version": "0.2.0",
			"services": gin.H{
				"mysql": "connected",
				"redis": "connected",
				"minio": "connected",
			},
		})
	})

	// 心跳
	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// API v1
	v1 := engine.Group("/api/v1")
	{
		// ========== Callback 接收（无需鉴权）==========
		v1.POST("/callback/crip", func(c *gin.Context) {
			// 读取 body
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				response.BadRequest(c, "读取请求体失败")
				return
			}

			// 解析 CRIP JSON
			var cb model.CRIPCallback
			if err := json.Unmarshal(body, &cb); err != nil {
				response.BadRequest(c, "JSON 解析失败: "+err.Error())
				return
			}

			// 校验必填字段
			if cb.SnowflakeID == "" {
				response.BadRequest(c, "缺少 snowflake_id")
				return
			}

			// 处理报警
			result, err := alarmSvc.ProcessCallback(c.Request.Context(), &cb)
			if err != nil {
				sugar.Errorf("处理 Callback 失败: %v", err)
				response.ServerError(c, "处理报警失败")
				return
			}

			sugar.Infof("Callback 处理完成: snowflake=%s, action=%s", cb.SnowflakeID, result.Action)
			response.Success(c, result)
		})

		// ========== 认证接口 ==========
		auth := v1.Group("/auth")
		{
			auth.POST("/login", func(c *gin.Context) {
				response.Success(c, gin.H{"message": "login — 待实现"})
			})
			auth.POST("/wechat/login", func(c *gin.Context) {
				response.Success(c, gin.H{"message": "wechat login — 待实现"})
			})
		}

		// ========== 手动补偿（管理员）==========
		v1.POST("/admin/compensate", func(c *gin.Context) {
			var req alarm.CompensateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				response.BadRequest(c, "参数错误: start_time 和 end_time 必填")
				return
			}
			result, err := alarmSvc.Compensate(c.Request.Context(), &req)
			if err != nil {
				sugar.Errorf("手动补偿失败: %v", err)
				response.ServerError(c, "手动补偿失败: "+err.Error())
				return
			}
			response.Success(c, result)
		})

		// ========== 工单中心（占位，后续实现）==========
		v1.GET("/work-orders/pending", func(c *gin.Context) {
			response.Success(c, gin.H{"list": []interface{}{}, "total": 0})
		})
	}

	sugar.Info("路由注册完成")
}
