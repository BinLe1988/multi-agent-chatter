package api

import (
	"multi-agent-chatter/api/handlers"
	"multi-agent-chatter/api/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置API路由
func SetupRouter(router *gin.Engine) {
	// 公共API
	public := router.Group("/api")
	{
		// 认证相关
		public.POST("/auth/login", handlers.Login)
		public.POST("/auth/register", handlers.Register)

		// 支付回调
		public.POST("/payments/callback", handlers.HandlePaymentCallback)
	}

	// 需要认证的API
	authorized := router.Group("/api")
	authorized.Use(middleware.Auth())
	{
		// 用户相关
		authorized.GET("/user", handlers.GetCurrentUser)
		authorized.PUT("/user/profile", handlers.UpdateUserProfile)
		authorized.POST("/auth/logout", handlers.Logout)

		// 订阅相关
		authorized.GET("/subscriptions", handlers.GetSubscriptionPlans)
		authorized.POST("/subscriptions", handlers.UpdateSubscription)

		// 充值相关
		authorized.GET("/recharge/packages", handlers.GetRechargePackages)
		authorized.POST("/recharge", handlers.CreateRechargeOrder)
		authorized.GET("/payments", handlers.GetPaymentHistory)
		authorized.GET("/payments/:orderNo", handlers.CheckPaymentStatus)

		// 聊天相关
		authorized.GET("/chat/sessions", handlers.GetChatSessions)
		authorized.POST("/chat/sessions", handlers.CreateChatSession)
		authorized.GET("/chat/sessions/:sessionId/messages", handlers.GetChatMessages)
		authorized.POST("/chat/messages", handlers.SendChatMessage)

		// 匹配相关
		authorized.POST("/matching", handlers.RequestMatching)
		authorized.DELETE("/matching", handlers.CancelMatching)
	}
}
