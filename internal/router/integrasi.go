package router

import (
	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/gin-gonic/gin"
)

func (r *Router) integrasiRoutes(version *gin.RouterGroup) {
	// URL Config - Protected with JWT authentication
	urlConfig := version.Group("/url-config")
	urlConfig.Use(r.jwtMw.RequireAuth()) // JWT protection for all URL Config routes
	{
		urlConfig.POST("", r.validMw.ValidateRequestBody(func() interface{} { return &dto.URLConfigRequest{} }), r.IntegrasiHandler.CreateURLConfig)
		urlConfig.PUT("/:id", r.validMw.ValidateRequestBody(func() interface{} { return &dto.URLConfigRequest{} }), r.IntegrasiHandler.UpdateURLConfig)
		urlConfig.DELETE("/:id", r.IntegrasiHandler.DeleteURLConfig)
		urlConfig.GET("/:id", r.IntegrasiHandler.GetByIDURLConfig)
		urlConfig.GET("", r.IntegrasiHandler.GetAllURLConfig)
	}

	// Path Config - Protected with JWT authentication
	pathConfig := version.Group("/path-config")
	pathConfig.Use(r.jwtMw.RequireAuth()) // JWT protection for all Path Config routes
	{
		pathConfig.POST("", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIConfigRequest{} }), r.IntegrasiHandler.CreateConfig)
		pathConfig.PUT("/:id", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIConfigRequest{} }), r.IntegrasiHandler.UpdateConfig)
		pathConfig.DELETE("/:id", r.IntegrasiHandler.DeleteConfig)
		pathConfig.GET("/:id", r.IntegrasiHandler.GetByIDConfig)
		pathConfig.GET("", r.IntegrasiHandler.GetAllConfig)
	}



	// integrasi := version.Group("/integrasi")
	// {
	// 	//Group
	// 	integrasi.POST("/group", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIGroupRequest{} }), r.IntegrasiHandler.CreateGroup)
	// 	integrasi.PUT("/group/:id", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIGroupRequest{} }), r.IntegrasiHandler.UpdateGroup)
	// 	integrasi.DELETE("/group/:id", r.IntegrasiHandler.DeleteGroup)
	// 	integrasi.GET("/group/:id", r.IntegrasiHandler.GetByIDGroup)
	// 	integrasi.GET("/group", r.IntegrasiHandler.GetAllGroup)

	// 	//Group Step
	// 	integrasi.POST("/group-step", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIGroupStepRequest{} }), r.IntegrasiHandler.CreateGroupStep)
	// 	integrasi.PUT("/group-step/:id", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIGroupStepRequest{} }), r.IntegrasiHandler.UpdateGroupStep)
	// 	integrasi.DELETE("/group-step/:id", r.IntegrasiHandler.DeleteGroupStep)
	// 	integrasi.GET("/group-step/:id", r.IntegrasiHandler.GetByIDGroupStep)
	// 	integrasi.GET("/group-step", r.IntegrasiHandler.GetAllGroupStep)

	// 	//Group Cron
	// 	integrasi.POST("/group-cron", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIGroupCronRequest{} }), r.IntegrasiHandler.CreateGroupCron)
	// 	integrasi.PUT("/group-cron/:id", r.validMw.ValidateRequestBody(func() interface{} { return &dto.APIGroupCronRequest{} }), r.IntegrasiHandler.UpdateGroupCron)
	// 	integrasi.DELETE("/group-cron/:id", r.IntegrasiHandler.DeleteGroupCron)
	// 	integrasi.GET("/group-cron/:id", r.IntegrasiHandler.GetByIDGroupCron)
	// 	integrasi.GET("/group-cron", r.IntegrasiHandler.GetAllGroupCron)

	// 	//Fungsi
	// 	integrasi.GET("/external/:slug", r.IntegrasiHandler.ExternalIntegrasi)
	// 	integrasi.GET("/external/:slug/:id", r.IntegrasiHandler.ExternalIntegrasi)
	// 	integrasi.POST("/external/:slug", r.IntegrasiHandler.ExternalIntegrasi)
	// 	integrasi.PUT("/external/:slug/:id", r.IntegrasiHandler.ExternalIntegrasi)
	// 	integrasi.DELETE("/external/:slug/:id", r.IntegrasiHandler.ExternalIntegrasi)
	// 	integrasi.PATCH("/external/:slug/:id", r.IntegrasiHandler.ExternalIntegrasi)

	// 	integrasi.POST("/execute/:slug", r.IntegrasiHandler.ExecuteBySlug)
	// }

}
