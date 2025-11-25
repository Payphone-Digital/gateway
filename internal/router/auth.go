package router

import "github.com/gin-gonic/gin"

func (r *Router) authRoutes(version *gin.RouterGroup) {
	auth := version.Group("/auth")
	{
		// Public routes (no authentication required)
		auth.POST("/login", r.authHandler.Login)
		auth.POST("/refresh", r.authHandler.RefreshToken)

		// Protected routes (JWT authentication required)
		protected := auth.Group("")
		protected.Use(r.jwtMw.RequireAuth())
		{
			// Logout user
			protected.POST("/logout", r.authHandler.Logout)
		}
	}
}