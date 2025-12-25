package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"member_API/auth"
	"member_API/controllers"
	"member_API/graphql"
)

// SetupRouter registers API routes on the provided Gin engine.
func SetupRouter(Router *gin.Engine) {
	Router.GET("/Hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello, RESTful API!"})
	})

	// Public - no authentication required
	public := Router.Group("/api/v1")
	{
		// Authentication-related routes
		public.POST("/register", controllers.Register)
		public.POST("/login", controllers.Login)
	}

	// GraphQL endpoint
	Router.Any("/graphql", func(c *gin.Context) {
		graphqlHandler := graphql.GetHandler()
		if graphqlHandler == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "GraphQL handler not initialized"})
			return
		}
		graphqlHandler.ServeHTTP(c.Writer, c.Request)
	})

	// Protected routes - require authentication
	protected := Router.Group("/api/v1")
	protected.Use(auth.AuthMiddleware()) // Add authentication middleware
	{
		protected.GET("/users", controllers.GetUsers)
		protected.GET("/user/:id", func(c *gin.Context) {
			controllers.GetUserByID(c)
		})
		protected.GET("/profile", controllers.GetProfile) // Get current user information
		protected.DELETE("/user/:id", controllers.DeleteUserByID)

		// Product routes
		protected.GET("/products", controllers.GetProducts)
		protected.GET("/product/:id", controllers.GetProductByID)
		protected.POST("/product", controllers.CreateProduct)
		protected.PUT("/product/:id", controllers.UpdateProduct)
		protected.DELETE("/product/:id", controllers.DeleteProduct)
	}
}
