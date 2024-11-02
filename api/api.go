package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/zhitoo/go-api/requests"
	"github.com/zhitoo/go-api/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

const maxRedisValueSize = 100 * 1024 // 100KB
const fileTrackingZSet = "file_cache_tracker"

type ApiError struct {
	Message string
}

type APIServer struct {
	listenAddr string
	storage    storage.Storage
	validator  *requests.Validator
	rdb        *redis.Client
}

func NewAPIServer(listenAddr string, storage storage.Storage, validator *requests.Validator, rdb *redis.Client) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		storage:    storage,
		validator:  validator,
		rdb:        rdb,
	}
}

func (s *APIServer) Run() {
	app := fiber.New(fiber.Config{
		Prefork: true,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Set("Accept", "application/json")
		// Go to next middleware:
		return c.Next()
	})

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(etag.New())
	app.Use(compress.New())
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))
	app.Use(securityHeaders)

	// Routes
	app.Post("/register", s.registerOriginServer)
	app.Get("/*", s.serveStatic)
	app.Listen(s.listenAddr)
}

func securityHeaders(c *fiber.Ctx) error {
	c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Content-Security-Policy", "default-src 'self'")
	return c.Next()
}

func cleanUpCache(rdb *redis.Client) {
	ctx := context.Background()

	now := time.Now().Unix()

	// Get all file paths with expiration time less than or equal to now
	expiredFiles, err := rdb.ZRangeByScore(ctx, fileTrackingZSet, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		log.Printf("Error fetching expired files from ZSet: %v", err)
		return
	}

	for _, filePath := range expiredFiles {
		// Delete the file from disk
		err := os.Remove(filePath)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("Error deleting file %s: %v", filePath, err)
			continue
		}

		// Remove the file path from the sorted set
		_, err = rdb.ZRem(ctx, fileTrackingZSet, filePath).Result()
		if err != nil {
			log.Printf("Error removing file %s from ZSet: %v", filePath, err)
		}
	}
}

func (s *APIServer) StartCacheCleaner() {
	ticker := time.NewTicker(60 * time.Minute)
	go func() {
		for range ticker.C {
			cleanUpCache(s.rdb)
		}
	}()
}
