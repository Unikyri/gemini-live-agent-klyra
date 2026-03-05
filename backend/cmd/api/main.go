package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
	httphandlers "github.com/Unikyri/gemini-live-agent-klyra/backend/internal/handlers/http"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/repositories"
)

func main() {
	// Load environment variables from .env file in local development.
	// In production (Cloud Run), these are set via Secret Manager or env config.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading environment variables from system")
	}

	// --- Database connection ---
	db, err := connectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established successfully.")

	// Auto-migrate domain models. For production, use SQL files in /migrations.
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Course{},
		&domain.Topic{},
		&domain.Material{},
		&domain.MaterialChunk{}, // US8: pgvector RAG store
	); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// --- Dependency Injection (Composition Root) ---
	// Only this file knows about concrete implementations.
	// Use cases and handlers only see interfaces (ports).

	jwtSvc := repositories.NewJWTService(
		mustEnv("JWT_SECRET"),
		mustEnv("REFRESH_TOKEN_SECRET"),
	)

	// --- Auth wiring ---
	userRepo := repositories.NewPostgresUserRepository(db)
	googleVerifier := repositories.NewGoogleVerifier(mustEnv("GOOGLE_CLIENT_ID"))
	authUseCase := usecases.NewAuthUseCase(userRepo, jwtSvc, googleVerifier)

	// --- Course wiring ---
	courseRepo := repositories.NewPostgresCourseRepository(db)
	topicRepo := repositories.NewPostgresTopicRepository(db)
	storageSvc := repositories.NewGCSStorageService()    // real GCS — uses GOOGLE_APPLICATION_CREDENTIALS
	imageGenSvc := repositories.NewVertexImagenService() // Imagen 3 on Vertex AI
	courseUseCase := usecases.NewCourseUseCase(courseRepo, topicRepo, storageSvc, imageGenSvc)

	// --- Material wiring (US4) ---
	materialRepo := repositories.NewPostgresMaterialRepository(db)
	textExtractor := repositories.NewPlainTextExtractor()
	materialUseCase := usecases.NewMaterialUseCase(materialRepo, topicRepo, courseRepo, storageSvc, textExtractor)

	// --- RAG wiring (US8) ---
	// SECURITY: EMBEDDING_CREDENTIALS_FILE should be the same service account used
	// for Imagen/GCS (GOOGLE_APPLICATION_CREDENTIALS). Separate if least-privilege needed.
	embeddingSvc, err := repositories.NewVertexEmbeddingService(
		mustEnv("GCP_PROJECT_ID"),
		getEnv("EMBEDDING_LOCATION", "us-central1"),
		getEnv("EMBEDDING_MODEL_ID", "text-embedding-004"),
		mustEnv("GOOGLE_APPLICATION_CREDENTIALS"),
	)
	if err != nil {
		log.Fatalf("Failed to initialise Vertex AI Embedding service: %v", err)
	}
	defer embeddingSvc.Close() // graceful shutdown of gRPC client
	chunkRepo := repositories.NewPostgresChunkRepository(db)
	ragUseCase := usecases.NewRAGUseCase(materialRepo, chunkRepo, embeddingSvc)

	// --- HTTP Router setup ---
	// BLOCKER fix: use gin.New() instead of gin.Default() to avoid trusting all proxies.
	// SetTrustedProxies(nil) disables proxy trust entirely (safe for Cloud Run).
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}

	// WARNING fix: security headers middleware.
	// These headers harden the API against common browser-based attacks.
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff") // prevent MIME sniffing
		c.Header("X-Frame-Options", "DENY")           // prevent clickjacking
		c.Header("X-XSS-Protection", "1; mode=block") // legacy XSS filter
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	// WARNING fix: CORS — only allow the configured origin.
	// In development this is localhost; in production set ALLOWED_ORIGIN in Cloud Run.
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "http://localhost:3000")
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint — used by Cloud Run and load balancers.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")

	// WARNING fix: rate limiter for sensitive auth endpoints (5 req/sec, burst 10 per IP).
	authRateLimiter := newIPRateLimiter(rate.Limit(5), 10)

	// Public routes (no JWT required)
	authHandler := httphandlers.NewAuthHandler(authUseCase)
	authHandler.RegisterRoutes(v1, authRateLimiter.RateLimit())

	// Protected routes — JWT middleware enforces authentication on all sub-routes.
	protected := v1.Group("/")
	protected.Use(httphandlers.AuthMiddleware(jwtSvc))
	{
		courseHandler := httphandlers.NewCourseHandler(courseUseCase)
		courseHandler.RegisterRoutes(protected)

		// US4 — Material upload endpoints (nested under courses/topics).
		materialHandler := httphandlers.NewMaterialHandler(materialUseCase)
		materialHandler.RegisterRoutes(protected)

		// US8 — RAG context retrieval endpoint.
		ragHandler := httphandlers.NewRAGHandler(ragUseCase)
		ragHandler.RegisterRoutes(protected)
	}

	port := getEnv("PORT", "8080")
	log.Printf("Klyra Backend starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// --- Rate Limiter (per IP) ---

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{limiters: make(map[string]*rate.Limiter), r: r, b: b}
}

func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	if l, ok := i.limiters[ip]; ok {
		return l
	}
	l := rate.NewLimiter(i.r, i.b)
	i.limiters[ip] = l
	return l
}

// RateLimit returns a Gin middleware that limits requests per client IP.
func (i *ipRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !i.getLimiter(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

// connectDB builds the PostgreSQL connection from environment variables.
func connectDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		mustEnv("DB_HOST"),
		mustEnv("DB_USER"),
		mustEnv("DB_PASSWORD"),
		mustEnv("DB_NAME"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_SSL_MODE", "disable"),
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// mustEnv retrieves a required environment variable or panics on startup.
// Failing fast on startup is safer than failing silently in production.
func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Required environment variable %s is not set. Check your .env file or Cloud Run configuration.", key)
	}
	return val
}

// getEnv retrieves an optional environment variable with a fallback default.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
