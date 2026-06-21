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
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/LeeJiangNan/WDOS/internal/model"
	"github.com/LeeJiangNan/WDOS/internal/pkg/config"
	"github.com/LeeJiangNan/WDOS/internal/pkg/logger"
	"github.com/LeeJiangNan/WDOS/internal/service/alarm"
	"github.com/LeeJiangNan/WDOS/internal/service/auth"
	"github.com/LeeJiangNan/WDOS/internal/service/notify"
	"github.com/LeeJiangNan/WDOS/internal/service/schedule"
	"github.com/LeeJiangNan/WDOS/internal/service/sla"
	"github.com/LeeJiangNan/WDOS/internal/service/workorder"
	"github.com/gorilla/websocket"
	jwtpkg "github.com/LeeJiangNan/WDOS/internal/pkg/jwt"
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
	jwtMgr := jwtpkg.New(cfg.JWT.Secret, cfg.JWT.ExpireSeconds)
	authSvc := auth.New(db, jwtMgr, cfg.Wechat.AppID, cfg.Wechat.AppSecret, sugar)
	alarmSvc := alarm.New(db, rdb, minioClient, cfg.MinIO.Bucket, cfg.Redis.Prefix, cfg.CRIP, sugar)
	templateSvc := workorder.NewTemplateService(db, sugar)
	orderSvc := workorder.NewService(db, sugar)
	notifyHub := notify.NewHub(db, sugar)
	scheduleSvc := schedule.New(db, sugar)

	// 8.5 初始化种子数据（管理员账号）
	seedAdmin(db, sugar)

	// 8.6 启动 SLA 引擎
	slaEngine := sla.New(db,
		cfg.SLA.AcceptL1Seconds, cfg.SLA.AcceptL2Seconds, cfg.SLA.AcceptL3Seconds,
		cfg.SLA.ProcessL1Seconds, cfg.SLA.ProcessL2Seconds, cfg.SLA.ProcessL3Seconds,
		sugar,
	)
	go slaEngine.Run(context.Background(), 1*time.Second)

	// 9. 注册路由
	registerRoutes(engine, alarmSvc, authSvc, templateSvc, orderSvc, scheduleSvc, notifyHub, jwtMgr, cfg, sugar)

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
	authSvc *auth.Service,
	templateSvc *workorder.TemplateService,
	orderSvc *workorder.Service,
	scheduleSvc *schedule.Service,
	notifyHub *notify.Hub,
	jwtMgr *jwtpkg.Manager,
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

	// WebSocket 实时通知
	var wsUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	engine.GET("/ws/notifications", func(c *gin.Context) {
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil { return }
		notifyHub.Register(conn, 0)
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
		authGroup := v1.Group("/auth")
		{
			// 微信小程序登录
			authGroup.POST("/wechat/login", func(c *gin.Context) {
				var req auth.WechatLoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					response.BadRequest(c, "缺少 code 参数")
					return
				}
				result, err := authSvc.WechatLogin(req.Code)
				if err != nil {
					sugar.Warnf("微信登录失败: %v", err)
					response.Unauthorized(c, "登录失败: "+err.Error())
					return
				}
				response.Success(c, result)
			})

			// Web 管理后台登录
			authGroup.POST("/login", func(c *gin.Context) {
				var req auth.WebLoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					response.BadRequest(c, "缺少 username 和 password")
					return
				}
				result, err := authSvc.WebLogin(req.Username, req.Password)
				if err != nil {
					sugar.Warnf("Web 登录失败: %v", err)
					response.Unauthorized(c, "用户名或密码错误")
					return
				}
				response.Success(c, result)
			})

			// Token 刷新
			authGroup.POST("/refresh", func(c *gin.Context) {
				token := c.GetHeader("Authorization")
				if token == "" || len(token) < 8 {
					response.Unauthorized(c, "缺少 token")
					return
				}
				token = token[7:] // 去掉 "Bearer "
				result, err := authSvc.RefreshToken(token)
				if err != nil {
					response.Unauthorized(c, "token 刷新失败: "+err.Error())
					return
				}
				response.Success(c, result)
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

		// ========== 工单模板管理 ==========
		templates := v1.Group("/templates")
		{
			templates.GET("", func(c *gin.Context) {
				status := c.Query("status")
				page := 1
				size := 20
				list, total, err := templateSvc.List(status, page, size)
				if err != nil {
					response.ServerError(c, err.Error())
					return
				}
				response.Success(c, gin.H{"list": list, "total": total})
			})

			templates.GET("/:id", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				tpl, err := templateSvc.Get(id)
				if err != nil {
					response.NotFound(c, err.Error())
					return
				}
				response.Success(c, tpl)
			})

			templates.POST("", func(c *gin.Context) {
				var req struct {
					Name        string          `json:"name"`
					Description string          `json:"description"`
					FormSchema  json.RawMessage `json:"form_schema"`
					FlowID      uint64          `json:"flow_id"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					response.BadRequest(c, "参数错误")
					return
				}
				tpl, err := templateSvc.Create(req.Name, req.Description, req.FormSchema, req.FlowID)
				if err != nil {
					response.BadRequest(c, err.Error())
					return
				}
				response.Success(c, tpl)
			})

			templates.PUT("/:id", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				var req struct {
					Name        string          `json:"name"`
					Description string          `json:"description"`
					FormSchema  json.RawMessage `json:"form_schema"`
					FlowID      uint64          `json:"flow_id"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					response.BadRequest(c, "参数错误")
					return
				}
				tpl, err := templateSvc.Update(id, req.Name, req.Description, req.FormSchema, req.FlowID)
				if err != nil {
					response.BadRequest(c, err.Error())
					return
				}
				response.Success(c, tpl)
			})

			templates.POST("/:id/toggle", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				var req struct {
					IsActive bool `json:"is_active"`
				}
				c.ShouldBindJSON(&req)
				tpl, err := templateSvc.Toggle(id, req.IsActive)
				if err != nil {
					response.NotFound(c, err.Error())
					return
				}
				response.Success(c, tpl)
			})
		}

		// ========== 工单中心 ==========
		orders := v1.Group("/work-orders")
		{
			orders.GET("/pending", func(c *gin.Context) {
				role := c.GetString("role")
				userID, _ := strconv.ParseUint(c.GetString("user_id"), 10, 64)
				deptID, _ := strconv.ParseUint(c.GetString("department_id"), 10, 64)
				list, total, _ := orderSvc.ListByStatus("pending", role, userID, deptID, 1, 20)
				response.Success(c, gin.H{"list": list, "total": total})
			})
			orders.GET("/processing", func(c *gin.Context) {
				role := c.GetString("role")
				userID, _ := strconv.ParseUint(c.GetString("user_id"), 10, 64)
				deptID, _ := strconv.ParseUint(c.GetString("department_id"), 10, 64)
				list, total, _ := orderSvc.ListByStatus("processing", role, userID, deptID, 1, 20)
				response.Success(c, gin.H{"list": list, "total": total})
			})
			orders.GET("/completed", func(c *gin.Context) {
				role := c.GetString("role")
				userID, _ := strconv.ParseUint(c.GetString("user_id"), 10, 64)
				deptID, _ := strconv.ParseUint(c.GetString("department_id"), 10, 64)
				list, total, _ := orderSvc.ListByStatus("completed", role, userID, deptID, 1, 20)
				response.Success(c, gin.H{"list": list, "total": total})
			})
			orders.GET("/:id", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				order, logs, err := orderSvc.GetOrder(id)
				if err != nil { response.NotFound(c, err.Error()); return }
				response.Success(c, gin.H{"order": order, "logs": logs})
			})
			orders.POST("/:id/accept", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				userID, _ := strconv.ParseUint(c.GetString("user_id"), 10, 64)
				order, err := orderSvc.AcceptOrder(id, userID, c.GetString("name"))
				if err != nil { response.BadRequest(c, err.Error()); return }
				response.Success(c, order)
			})
			orders.POST("/:id/submit", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				userID, _ := strconv.ParseUint(c.GetString("user_id"), 10, 64)
				var req struct {
					Resolution string `json:"resolution"`
					FormData   string `json:"form_data"`
					ProofImages string `json:"proof_images"`
				}
				c.ShouldBindJSON(&req)
				order, err := orderSvc.SubmitOrder(id, userID, c.GetString("name"), req.Resolution, req.FormData, req.ProofImages)
				if err != nil { response.BadRequest(c, err.Error()); return }
				response.Success(c, order)
			})
			orders.POST("/:id/transfer", func(c *gin.Context) {
				id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
				var req struct {
					ToUserID   uint64 `json:"transfer_to_user_id"`
					ToUserName string `json:"transfer_to_user_name"`
					Reason     string `json:"reason"`
				}
				c.ShouldBindJSON(&req)
				order, err := orderSvc.TransferOrder(id, req.ToUserID, req.ToUserName, c.GetString("name"), req.Reason)
				if err != nil { response.BadRequest(c, err.Error()); return }
				response.Success(c, order)
			})
		}
		// ========== 排班管理 ==========
		schedules := v1.Group("/schedules")
		{
			schedules.GET("", func(c *gin.Context) {
				date := c.Query("date")
				if date == "" { date = time.Now().Format("2006-01-02") }
				result, _ := scheduleSvc.GetByDate(date, 0)
				response.Success(c, result)
			})
			schedules.POST("", func(c *gin.Context) {
				var req schedule.SetScheduleReq
				if err := c.ShouldBindJSON(&req); err != nil {
					response.BadRequest(c, "参数错误")
					return
				}
				result, err := scheduleSvc.SetSchedule(&req)
				if err != nil { response.BadRequest(c, err.Error()); return }
				response.Success(c, result)
			})
		}
	}

	sugar.Info("路由注册完成")
}

// seedAdmin 初始化管理员账号（幂等，已存在则跳过）
func seedAdmin(db *gorm.DB, sugar *zap.SugaredLogger) {
	var count int64
	db.Model(&model.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	hashedPwd, err := auth.HashPassword("Admin@123")
	if err != nil {
		sugar.Errorf("创建管理员密码失败: %v", err)
		return
	}

	admin := &model.User{
		Name:     "admin",
		Phone:    "00000000000",
		Password: hashedPwd,
		Role:     "admin",
		Status:   "active",
	}
	if err := db.Create(admin).Error; err != nil {
		sugar.Errorf("创建管理员账号失败: %v", err)
		return
	}

	// 创建默认部门
	dept := &model.Department{Name: "管理部"}
	db.FirstOrCreate(dept, model.Department{Name: "管理部"})
	db.FirstOrCreate(&model.UserGroup{Name: "管理员组", DepartmentID: dept.ID})

	sugar.Info("已初始化管理员账号: admin / Admin@123")
}
