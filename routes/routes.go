package routes

import (
	"github.com/gin-gonic/gin"

	"member_API/auth"
	"member_API/controllers"
)

func SetupRouter(Router *gin.Engine) {
	Router.GET("/Hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello, RESTful API!"})
	})

	public := Router.Group("/api/v1")
	{
		public.POST("/register", controllers.Register)
		public.POST("/login", controllers.Login)
	}

	protected := Router.Group("/api/v1")
	protected.Use(auth.AuthMiddleware())
	{
		protected.GET("/users", controllers.GetUsers)
		protected.GET("/user/:id", func(c *gin.Context) {
			controllers.GetUserByID(c)
		})
		protected.GET("/profile", controllers.GetProfile)
		protected.DELETE("/user/:id", controllers.DeleteUserByID)
		protected.POST("/auth/regenerate-key", controllers.RegenerateAPIKey)

		// 通知提供者 API
		protected.POST("/providers", controllers.CreateProvider)
		protected.GET("/providers", controllers.GetProviders)
		protected.GET("/providers/:id", controllers.GetProvider)
		protected.PUT("/providers/:id", controllers.UpdateProvider)
		protected.DELETE("/providers/:id", controllers.DeleteProvider)
		protected.GET("/providers/:id/test", controllers.TestProvider)
	}

	apiKeyProtected := Router.Group("/api/v1")
	apiKeyProtected.Use(auth.APIKeyMiddleware())
	{
		apiKeyProtected.GET("/auth/verify-key", controllers.VerifyAPIKey)
	}
}
