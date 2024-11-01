package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/h2non/bimg"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
	"github.com/zhitoo/go-api/requests"
	"github.com/zhitoo/go-api/storage"
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type ApiError struct {
	Message string
}

type APIServer struct {
	listenAddr string
	storage    storage.Storage
	validator  *requests.Validator
}

func NewAPIServer(listenAddr string, storage storage.Storage, validator *requests.Validator) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		storage:    storage,
		validator:  validator,
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

	// Start server
	log.Fatal(app.Listen(":8080"))

	log.Println("JSON API running on port: ", s.listenAddr)
	app.Listen(s.listenAddr)
}

func securityHeaders(c *fiber.Ctx) error {
	c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Content-Security-Policy", "default-src 'self'")
	return c.Next()
}

func (s *APIServer) serveStatic(c *fiber.Ctx) error {
	rdb := redis.NewClient(&redis.Options{})
	ctx := context.Background()

	path := filepath.Clean(c.Path())
	//hostname := c.Hostname()

	// Split the path to extract the site identifier and resource path
	segments := strings.SplitN(path, "/", 3) // ["", "siteIdentifier", "resourcePath"]
	if len(segments) < 3 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid URL format")
	}
	siteIdentifier := segments[1]
	resourcePath := "/" + segments[2]

	// Create cache key
	cacheKey := siteIdentifier + ":" + resourcePath

	// Include query parameters in cacheKey for resized images
	widthStr := c.Query("width")
	heightStr := c.Query("height")
	if widthStr != "" || heightStr != "" {
		cacheKey += fmt.Sprintf("?width=%s&height=%s", widthStr, heightStr)
	}

	// Check Redis cache
	cachedContent, err := rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		contentType := getContentType(resourcePath, cachedContent)
		c.Set("Content-Type", contentType)
		return c.Send(cachedContent)
	}

	// Retrieve the origin server URL from the database using siteIdentifier
	origin, _ := s.storage.GetOriginServerBySiteIdentifier(siteIdentifier)
	if origin.ID == 0 {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Origin server not configured")
		}
	}

	// Construct the origin URL
	originURL := origin.OriginURL + resourcePath

	// Implement locking to prevent cache stampede
	locked, err := acquireLock(cacheKey, 30*time.Second)
	if err != nil {
		log.Printf("Error acquiring lock: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	if locked {
		defer releaseLock(cacheKey)

		// Fetch, process, and cache the content
		return fetchProcessAndCacheContent(c, originURL, cacheKey, resourcePath, widthStr, heightStr)
	} else {
		// Wait and retry logic with a maximum retry limit
		retries := 0
		maxRetries := 5
		for retries < maxRetries {
			time.Sleep(200 * time.Millisecond)
			retries++
			return s.serveStatic(c)
		}
		return c.Status(fiber.StatusServiceUnavailable).SendString("Service Unavailable")
	}
}

func fetchProcessAndCacheContent(c *fiber.Ctx, originURL, cacheKey, path, widthStr, heightStr string) error {

	rdb := redis.NewClient(&redis.Options{})
	ctx := context.Background()

	// Fetch from the origin server
	resp, err := http.Get(originURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Error fetching from origin: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("File Not Found")
	}
	defer resp.Body.Close()

	// Read the content
	fileContent, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading origin response: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	// Determine the content type
	contentType := getContentType(path, fileContent)
	c.Set("Content-Type", contentType)

	// Process content based on type
	if contentType == "text/css" || contentType == "application/javascript" {
		// Minify CSS or JS
		fileContent = minifyContent(contentType, fileContent)
	} else if isImage(contentType) && (widthStr != "" || heightStr != "") {
		// Resize image
		fileContent, err = resizeImage(fileContent, widthStr, heightStr)
		if err != nil {
			log.Printf("Error resizing image: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Image Processing Error")
		}
	}

	// Cache the content in Redis
	if err := rdb.Set(ctx, cacheKey, fileContent, 10*time.Minute).Err(); err != nil {
		log.Printf("Error caching content in Redis: %v", err)
	}

	// Serve the content
	return c.Send(fileContent)
}

func getContentType(path string, content []byte) string {
	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}
	return mimeType
}

func minifyContent(contentType string, content []byte) []byte {
	minifiedContent, err := m.Bytes(contentType, content)
	if err != nil {
		log.Printf("Error minifying content: %v", err)
		return content // Fallback to original content
	}
	return minifiedContent
}

func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

func resizeImage(imageData []byte, widthStr, heightStr string) ([]byte, error) {
	// Parse width and height
	width, _ := strconv.Atoi(widthStr)
	height, _ := strconv.Atoi(heightStr)

	// Validate dimensions
	const maxDimension = 2000
	if width > maxDimension || height > maxDimension {
		return nil, errors.New("requested dimensions are too large")
	}

	// Set up image processing options
	options := bimg.Options{
		Width:  width,
		Height: height,
	}

	// Process the image
	newImage, err := bimg.NewImage(imageData).Process(options)
	if err != nil {
		return nil, err
	}
	return newImage, nil
}

func acquireLock(key string, ttl time.Duration) (bool, error) {
	rdb := redis.NewClient(&redis.Options{})
	ctx := context.Background()
	return rdb.SetNX(ctx, "lock:"+key, 1, ttl).Result()
}

func releaseLock(key string) {
	rdb := redis.NewClient(&redis.Options{})
	ctx := context.Background()
	rdb.Del(ctx, "lock:"+key)
}

var m *minify.M

func init() {
	// Initialize Minifier
	m = minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
}
