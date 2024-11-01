package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/h2non/bimg"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
	"gorm.io/gorm"
)



func (s *APIServer) serveStatic(c *fiber.Ctx) error {
	rdb := s.rdb
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
	cachedValue, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		// Determine if cachedValue is a file path or content
		if strings.HasPrefix(cachedValue, "file:") {
			// It's a file path
			filePath := cachedValue[5:] // Remove "file:" prefix
			return c.SendFile(filePath)
		} else {
			// It's content stored directly in Redis
			content := []byte(cachedValue)
			contentType := getContentType(resourcePath, content)
			c.Set("Content-Type", contentType)
			return c.Send(content)
		}
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
	locked, err := acquireLock(rdb, cacheKey, 30*time.Second)
	if err != nil {
		log.Printf("Error acquiring lock: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	if locked {
		defer releaseLock(rdb, cacheKey)

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

	// Decide whether to store content in Redis or on disk
	if len(fileContent) <= maxRedisValueSize {
		// Store content directly in Redis
		if err := rdb.Set(ctx, cacheKey, fileContent, expireTimeInMinute*time.Minute).Err(); err != nil {
			log.Printf("Error caching content in Redis: %v", err)
		}
		// Serve the content
		return c.Send(fileContent)
	} else {
		// Store file on disk
		filePath, err := saveFileToDisk(cacheKey, fileContent)
		if err != nil {
			log.Printf("Error saving file to disk: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
		// Store file path in Redis
		if err := rdb.Set(ctx, cacheKey, "file:"+filePath, expireTimeInMinute).Err(); err != nil {
			log.Printf("Error caching file path in Redis: %v", err)
		}
		// Add entry to the sorted set with expiration timestamp
		expiration := time.Now().Add(expireTimeInMinute)
		if err := rdb.ZAdd(ctx, fileTrackingZSet, &redis.Z{
			Score:  float64(expiration.Unix()),
			Member: filePath,
		}).Err(); err != nil {
			log.Printf("Error adding file to tracking ZSet: %v", err)
		}
		// Serve the file
		return c.SendFile(filePath)
	}
}

func saveFileToDisk(cacheKey string, content []byte) (string, error) {
	// Define the base directory for cached files
	baseDir := "./.cache"

	// Ensure the base directory exists
	err := os.MkdirAll(baseDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	// Generate a safe file name based on the cache key
	fileName := generateSafeFileName(cacheKey)

	// Full path to the cached file
	filePath := filepath.Join(baseDir, fileName)

	// Write the content to the file
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func generateSafeFileName(cacheKey string) string {
	// Use a hash function to generate a unique file name
	h := sha256.New()
	h.Write([]byte(cacheKey))
	fileName := hex.EncodeToString(h.Sum(nil))
	return fileName
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

func acquireLock(rdb *redis.Client, key string, ttl time.Duration) (bool, error) {
	ctx := context.Background()
	return rdb.SetNX(ctx, "lock:"+key, 1, ttl).Result()
}

func releaseLock(rdb *redis.Client, key string) {
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
