package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"task-management/backend/internal/config"
	"task-management/backend/internal/handler"
	appmiddleware "task-management/backend/internal/middleware"
	"task-management/backend/internal/repository"
	"task-management/backend/internal/service"
)

func main() {
	cfg := config.Load()
	db, err := sql.Open("mysql", cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect database: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddress, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Redis unavailable at startup; requests will use MySQL: %v", err)
	}

	taskRepository := repository.NewMySQLTaskRepository(db)
	taskService := service.NewTaskService(taskRepository, service.NewRedisCache(redisClient))
	taskHandler := handler.NewTaskHandler(taskService)

	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger(), appmiddleware.ErrorLogger())
	router.Use(cors.New(cors.Config{AllowOrigins: []string{"*"}, AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Content-Type"}, MaxAge: 12 * time.Hour}))
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	taskHandler.Register(router.Group("/api"))

	log.Printf("API listening on :%s", cfg.AppPort)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("run API: %v", err)
	}
}
