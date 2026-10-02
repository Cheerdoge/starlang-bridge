package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"starlang-bridge/handler"
	"starlang-bridge/server"
)

func main() {
	cfg := server.LoadConfig()

	db, err := server.InitDB(cfg.MySQL)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer func() {
		if err := server.CloseDB(db); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}()

	jwtManager := server.NewJWTManager(cfg.JWT)
	userService := server.NewUserService(db)
	childService := server.NewChildService(db)
	trainingService := server.NewTrainingService(db)
	wechatClient := server.NewWeChatClient(cfg.WeChat)

	scenarioService, err := server.LoadScenarios(cfg.ScenarioFile)
	if err != nil {
		log.Fatalf("加载情境配置失败: %v", err)
	}

	evaluator := server.NewEvaluator(cfg.AI)
	dialogueService := server.NewDialogueService(trainingService, scenarioService, childService, evaluator)

	userHandler := handler.NewUserHandler(userService, wechatClient, jwtManager)
	childHandler := handler.NewChildHandler(childService)
	scenarioHandler := handler.NewScenarioHandler(scenarioService)
	trainingHandler := handler.NewTrainingHandler(dialogueService)
	taskCardHandler := handler.NewTaskCardHandler(trainingService)

	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/auth/login", userHandler.Login)
		api.GET("/scenarios", scenarioHandler.List)

		auth := api.Group("", handler.AuthMiddleware(jwtManager))
		{
			auth.GET("/user/me", userHandler.Me)
			auth.POST("/user/consent", userHandler.UpdateConsent)

			auth.GET("/user/child", childHandler.Get)
			auth.PUT("/user/child", childHandler.Upsert)

			auth.GET("/scenarios/:id", scenarioHandler.Get)

			auth.POST("/training/sessions", trainingHandler.Start)
			auth.GET("/training/sessions/:id", trainingHandler.Detail)
			auth.POST("/training/sessions/:id/answers", trainingHandler.Answer)
			auth.POST("/training/sessions/:id/abort", trainingHandler.Abort)

			auth.GET("/task-cards", taskCardHandler.List)
			auth.POST("/task-cards/:id/backfill", taskCardHandler.Backfill)
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("星语桥服务启动，环境=%s，端口=%s", cfg.Env, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务关闭异常: %v", err)
	}
	log.Println("服务已退出")
}
