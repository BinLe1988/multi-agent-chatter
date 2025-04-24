package main

import (
	"log"

	"github.com/BinLe1988/multi-agent-chatter/api"
	"github.com/BinLe1988/multi-agent-chatter/configs"
	"github.com/BinLe1988/multi-agent-chatter/database"
	"github.com/BinLe1988/multi-agent-chatter/pkg/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化JWT
	utils.InitJWT(cfg)

	// 初始化数据库连接
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 创建Gin实例
	router := gin.Default()

	// 设置路由
	api.SetupRouter(router)

	// 启动服务器
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
