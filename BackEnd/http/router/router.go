package router

import (
	"IslaMemory/BackEnd/config"
	"IslaMemory/BackEnd/http/middleware"
	"IslaMemory/BackEnd/internal/auth"
	"IslaMemory/BackEnd/internal/category"
	"IslaMemory/BackEnd/internal/user"
	"IslaMemory/BackEnd/internal/video"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Static("/static", "./storage")
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 初始化依赖
	passwordSvc := auth.NewPasswordService()
	jwtSvc := auth.NewJWTService(cfg.JWT)

	// 用户
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo, passwordSvc, jwtSvc)
	userHandler := user.NewHandler(userSvc)

	// 分类
	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categoryService)

	// 视频
	// video 模块依赖初始化
	videoRepo := video.NewRepository(db)
	videoService := video.NewService(videoRepo)
	videoHandler := video.NewHandler(videoService)

	api := r.Group("/api/v1")

	// 用户登录注册修改密码模块
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
	}

	userGroup := api.Group("/users")
	userGroup.Use(middleware.JWTAuth(jwtSvc, userSvc))
	{
		userGroup.GET("/me", userHandler.Me)
		userGroup.PUT("/me", userHandler.UpdateMe)
		userGroup.GET("/me/favorites", userHandler.MyFavorites)
		userGroup.GET("/me/uploads", userHandler.MyUploads)
		userGroup.POST("/me/password", userHandler.ChangePassword)
	}

	// 分类获取模块
	api.GET("/categories", categoryHandler.List)
	api.GET("/categories/:slug", categoryHandler.GetBySlug)

	// 管理员分类管理
	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.JWTAuth(jwtSvc, userSvc), middleware.AdminOnly())
	{
		adminGroup.GET("/categories/:id", categoryHandler.GetByID)
		adminGroup.GET("/categories", categoryHandler.AdminList)
		adminGroup.POST("/categories", categoryHandler.Create)
		adminGroup.PATCH("/categories/:id", categoryHandler.Update)
		adminGroup.DELETE("/categories/:id", categoryHandler.Delete)
		adminGroup.PATCH("/categories/:id/restore", categoryHandler.Restore)
	}

	// 视频模块
	// 公开视频接口
	api.GET("/videos", videoHandler.List)
	api.GET("/videos/:public_id", videoHandler.Detail)
	api.POST("/videos/:public_id/play", videoHandler.IncreasePlay)

	// 登录用户上视频接口
	uploadGroup := api.Group("/videos")
	uploadGroup.GET("/:public_id/comments", videoHandler.ListComments)
	uploadGroup.Use(middleware.JWTAuth(jwtSvc, userSvc))
	{
		uploadGroup.POST("/upload", videoHandler.Upload)
		uploadGroup.GET("/:public_id/interaction", videoHandler.InteractionState)
		// 点赞
		uploadGroup.POST("/:public_id/like", videoHandler.Like)
		uploadGroup.DELETE("/:public_id/like", videoHandler.Unlike)

		// 收藏
		uploadGroup.POST("/:public_id/favorite", videoHandler.Favorite)
		uploadGroup.DELETE("/:public_id/favorite", videoHandler.Unfavorite)

		// 评论（列表 + 发表评论/回复）
		uploadGroup.POST("/:public_id/comments", videoHandler.CreateComment)
	}

	return r
}
