package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppPort       string
	DatabaseDSN   string
	RedisAddress  string
	RedisPassword string
	RedisDB       int
}

func Load() Config {
	redisDB, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&clientFoundRows=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		env("DB_USER", "task_user"), env("DB_PASSWORD", "task_password"),
		env("DB_HOST", "localhost"), env("DB_PORT", "3306"), env("DB_NAME", "task_management"))
	return Config{
		AppPort:       env("APP_PORT", "8080"),
		DatabaseDSN:   dsn,
		RedisAddress:  env("REDIS_HOST", "localhost") + ":" + env("REDIS_PORT", "6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
