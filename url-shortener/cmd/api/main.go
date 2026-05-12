package main

import (
	"context"
	"log"

	"url-shortener/internal/api"
	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/db"
	"url-shortener/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	// 1. laod config
	godotenv.Load() // load .env in development
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(cfg.DatabaseURL, "file://migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	redisClient, err := cache.Connect(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	s := store.New(database.Pool, redisClient)
	r := api.NewRouter(s, cfg)

	log.Printf("starting server on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
