package router

import "github.com/gin-gonic/gin"

func (r *Router) userRoutes(version *gin.RouterGroup) {
	users := version.Group("/users")
	{
		// All user routes require JWT authentication
		users.Use(r.jwtMw.RequireAuth())
		{
			// Get all users with pagination and search
			users.GET("", r.userHandler.GetAll)

			// Get user by ID
			users.GET("/:id", r.userHandler.GetByID)

			// Create new user
			users.POST("", r.userHandler.CreateUser)

			// Update user information (first name, last name, phone - email cannot be changed)
			users.PUT("/:id", r.userHandler.UpdateUser)

			// Update user password with current password verification
			users.PUT("/:id/password", r.userHandler.UpdatePassword)

			// Soft delete user (with security validations)
			users.DELETE("/:id", r.userHandler.DeleteUser)
		}
	}
}
