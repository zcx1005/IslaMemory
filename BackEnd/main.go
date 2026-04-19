package main

import (
	"IslaMemory/BackEnd/config"
	"IslaMemory/BackEnd/http/middleware"
	"IslaMemory/BackEnd/http/router"
	"IslaMemory/BackEnd/internal/category"
	"IslaMemory/BackEnd/internal/comment"
	"IslaMemory/BackEnd/internal/favorite"
	"IslaMemory/BackEnd/internal/like"
	"IslaMemory/BackEnd/internal/user"
	"IslaMemory/BackEnd/internal/video"
	"IslaMemory/BackEnd/platform/cache"
	"IslaMemory/BackEnd/platform/db"
	"fmt"
	"log"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	mysqlDB, err := db.NewMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("connect mysql failed: %v", err)
	}

	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("connect redis failed: %v", err)
	}

	_ = redisClient

	err = mysqlDB.AutoMigrate(
		&user.User{},
		&category.Category{},
		&video.Video{},
		&like.VideoLike{},
		&favorite.VideoFavorite{},
		&comment.VideoComment{},
	)
	if err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	log.Println("mysql connected")
	log.Println("redis connected")
	log.Println("auto migrate success")

	r := router.New(cfg, mysqlDB)

	r.Use(middleware.CORS())

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	log.Printf("server start at %s", addr)

	for _, ri := range r.Routes() {
		log.Printf("%-6s http://127.0.0.1:%d%s", ri.Method, cfg.HTTP.Port, ri.Path)
	}

	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
