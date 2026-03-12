package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/usecases"
	httphandlers "github.com/Unikyri/gemini-live-agent-klyra/backend/internal/handlers/http"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/infrastructure/database"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/repositories"
)

func main() {
	// Load environment variables from .env file in local development.
	// In production (Cloud Run), these are set via Secret Manager or env config.
	// Overload environment variables to prioritize .env over system variables
	// Try loading from multiple locations: current dir, backend dir, parent dir
	envPaths := []string{".env", "backend/.env", "../backend/.env"}
	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Overload(path); err == nil {
			log.Printf("✓ Loaded environment from: %s", path)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Println("⚠ No .env file found, reading environment variables from system")
	}

	// --- Database connection (DB_MODE: local|cloud) ---
	dbRepo, err := initDBRepository()
	if err != nil {
		log.Fatalf("Failed to initialize database repository: %v", err)
	}
	defer func() {
		if err := dbRepo.Close(); err != nil {
			log.Printf("Database close warning: %v", err)
		}
	}()

	if err := dbRepo.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	// Release phase / one-off migration mode (Heroku release command).
	// When enabled, run migrations and exit without starting the HTTP server.
	if strings.EqualFold(os.Getenv("RUN_MIGRATIONS_ONLY"), "true") {
		log.Println("RUN_MIGRATIONS_ONLY=true — running database migrations and exiting")
		if err := dbRepo.RunMigrations("./migrations"); err != nil {
			log.Fatalf("Database migrations failed: %v", err)
		}
		log.Println("Database migrations completed successfully.")
		os.Exit(0)
	}

	// Backwards-compatible local behaviour: run migrations on boot unless explicitly disabled.
	// In production on Heroku, we set RUN_MIGRATIONS_ON_BOOT=false and rely on release phase.
	runMigrationsOnBoot := strings.ToLower(strings.TrimSpace(getEnv("RUN_MIGRATIONS_ON_BOOT", "true")))
	if runMigrationsOnBoot != "false" {
		if err := dbRepo.RunMigrations("./migrations"); err != nil {
			log.Fatalf("Database migrations failed: %v", err)
		}
	} else {
		log.Println("Skipping migrations on boot (RUN_MIGRATIONS_ON_BOOT=false)")
	}

	db := dbRepo.GetDB()
	log.Println("Database connection established successfully.")

	// Auto-migrate domain models. For production, use SQL files in /migrations.
	/*if err := db.AutoMigrate(
		&domain.User{},
		&domain.Course{},
		&domain.Topic{},
		&domain.Material{},
		&domain.MaterialChunk{}, // US8: pgvector RAG store
	); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}*/

	// --- Dependency Injection (Composition Root) ---
	// Only this file knows about concrete implementations.
	// Use cases and handlers only see interfaces (ports).

	isProduction := strings.EqualFold(getEnv("ENV", "development"), "production")

	// Guardrail: Heroku filesystem is ephemeral. Never allow local storage in production.
	if isProduction && strings.EqualFold(getEnv("STORAGE_MODE", "gcs"), "local") {
		log.Fatalf("FATAL: STORAGE_MODE=local is not supported in production (ephemeral filesystem). Use STORAGE_MODE=gcs.")
	}

	jwtSvc := repositories.NewJWTService(
		mustEnv("JWT_SECRET"),
		mustEnv("REFRESH_TOKEN_SECRET"),
	)

	// --- Auth wiring ---
	userRepo := repositories.NewPostgresUserRepository(db)
	googleVerifier := repositories.NewGoogleVerifier(mustEnv("GOOGLE_CLIENT_ID"))
	authStrategies := map[string]ports.AuthStrategy{
		"google": repositories.NewGoogleAuthStrategy(googleVerifier),
		"guest":  repositories.NewGuestAuthStrategy(),
	}
	authUseCase := usecases.NewAuthUseCase(userRepo, jwtSvc, googleVerifier, authStrategies)
	learningProfileUseCase := usecases.NewLearningProfileUseCase(userRepo)

	// --- Course wiring ---
	courseRepo := repositories.NewPostgresCourseRepository(db)
	topicRepo := repositories.NewPostgresTopicRepository(db)
	chunkRepo := repositories.NewPostgresChunkRepository(db)
	materialRepo := repositories.NewPostgresMaterialRepository(db)
	correctionRepo := repositories.NewPostgresCorrectionRepository(db)
	storageSvc := initStorageService()                   // selected by STORAGE_MODE
	imageGenSvc := repositories.NewVertexImagenService() // Imagen 3 on Vertex AI
	courseUseCase := usecases.NewCourseUseCaseWithCascade(courseRepo, topicRepo, materialRepo, chunkRepo, db, storageSvc, imageGenSvc)

	// --- RAG wiring (US8) ---
	// SECURITY: EMBEDDING_CREDENTIALS_FILE should be the same service account used
	// for Imagen/GCS (GOOGLE_APPLICATION_CREDENTIALS). Separate if least-privilege needed.
	// In development, GCP_PROJECT_ID and GOOGLE_APPLICATION_CREDENTIALS are optional
	var embeddingSvc ports.Embedder
	var ragUseCase *usecases.RAGUseCase

	gcpProjectID := getEnv("GCP_PROJECT_ID", "")
	googleAppCreds := getEnv("GOOGLE_APPLICATION_CREDENTIALS", "")

	if gcpProjectID != "" && googleAppCreds != "" {
		var err error
		embeddingSvc, err = repositories.NewVertexEmbeddingService(
			gcpProjectID,
			getEnv("EMBEDDING_LOCATION", "us-central1"),
			getEnv("EMBEDDING_MODEL_ID", "text-embedding-004"),
			googleAppCreds,
		)
		if err != nil {
			log.Printf("WARNING: Failed to initialise Vertex AI Embedding service (RAG will be disabled): %v", err)
			embeddingSvc = nil
		} else {
			// Safe cast to get Close() method for cleanup
			if svc, ok := embeddingSvc.(interface{ Close() error }); ok {
				defer svc.Close()
			}
		}
	} else {
		log.Println("WARNING: GCP_PROJECT_ID or GOOGLE_APPLICATION_CREDENTIALS not set. RAG (embeddings) will be disabled. Set these variables to enable Vector Search.")
		embeddingSvc = nil
	}

	ragUseCase = usecases.NewRAGUseCaseWithCorrections(materialRepo, chunkRepo, topicRepo, correctionRepo, embeddingSvc)
	summaryGenerator := repositories.NewMarkdownSummaryGenerator()
	topicUseCase := usecases.NewTopicUseCase(topicRepo, materialRepo, summaryGenerator)

	// --- Material wiring (US4) ---
	textExtractor := repositories.NewPlainTextExtractor()
	materialUseCase := usecases.NewMaterialUseCase(materialRepo, topicRepo, courseRepo, storageSvc, textExtractor, correctionRepo, ragUseCase)

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

	// WARNING fix: CORS — allow configured origins (comma-separated).
	// In development this can include web localhost ports.
	// Default: multiple common dev ports (React 3000, Vite 5173, Flutter web, custom development)
	allowedOriginsRaw := getEnv("ALLOWED_ORIGINS", getEnv("ALLOWED_ORIGIN", ""))
	if isProduction && strings.TrimSpace(allowedOriginsRaw) == "" {
		log.Fatalf("Required environment variable ALLOWED_ORIGINS is not set for production (ENV=production).")
	}
	if strings.TrimSpace(allowedOriginsRaw) == "" {
		allowedOriginsRaw = "http://localhost:3000,http://localhost:5173,http://localhost:5174,http://127.0.0.1:3000,http://127.0.0.1:5173,http://127.0.0.1:5174"
	}
	allowedOrigins := parseAllowedOrigins(allowedOriginsRaw)
	allowedOriginFunc := func(origin string) bool {
		for _, configured := range allowedOrigins {
			if strings.EqualFold(strings.TrimSpace(configured), strings.TrimSpace(origin)) {
				return true
			}
		}

		// Development-only fallback to allow localhost ports.
		if !isProduction {
			originLower := strings.ToLower(strings.TrimSpace(origin))
			return strings.HasPrefix(originLower, "http://localhost:") ||
				strings.HasPrefix(originLower, "http://127.0.0.1:") ||
				strings.HasPrefix(originLower, "https://localhost:") ||
				strings.HasPrefix(originLower, "https://127.0.0.1:")
		}
		return false
	}
	router.Use(cors.New(cors.Config{
		AllowOriginFunc:  allowedOriginFunc,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint — used by Cloud Run and load balancers.
	router.GET("/health", func(c *gin.Context) {
		check := strings.ToLower(strings.TrimSpace(c.Query("check")))
		if strings.Contains(check, "db") {
			if err := dbRepo.Ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "degraded",
					"db":     "unreachable",
					"error":  err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "connected"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if strings.EqualFold(getEnv("STORAGE_MODE", "gcs"), "local") {
		storagePath := getEnv("STORAGE_PATH", "./storage")
		router.Static("/static", storagePath)
		log.Printf("Local static file serving enabled at /static from %s", storagePath)
	}

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
		ragHandler := httphandlers.NewRAGHandlerWithCourseUseCase(ragUseCase, courseUseCase)
		ragHandler.RegisterRoutes(protected)

		// Sprint 7 — Topic readiness and summary endpoints.
		topicHandler := httphandlers.NewTopicHandlerWithCourseUseCase(topicUseCase, courseUseCase)
		topicHandler.RegisterRoutes(protected)

		// Sprint 8 — Learning profile endpoints (feature-flagged).
		lpHandler := httphandlers.NewLearningProfileHandler(learningProfileUseCase)
		lpHandler.RegisterRoutes(protected)
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
// Excludes OPTIONS requests (CORS preflight) from rate limiting to allow proper CORS handling.
func (i *ipRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow OPTIONS requests (CORS preflight) without rate limiting
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		if !i.getLimiter(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

func initDBRepository() (ports.DBRepository, error) {
	// Precedence 1: DATABASE_URL (Heroku, Render, Railway, etc.)
	if databaseURL := os.Getenv("DATABASE_URL"); strings.TrimSpace(databaseURL) != "" {
		log.Printf("Database mode: url (DATABASE_URL detected)")
		return database.NewPostgreSQLRepositoryFromURL(databaseURL)
	}

	dbMode := strings.ToLower(getEnv("DB_MODE", "local"))
	log.Printf("Database mode: %s", dbMode)

	if dbMode == "cloud" {
		return database.NewCloudSQLRepository(
			getEnv("DB_INSTANCE_CONNECTION_NAME", getEnv("INSTANCE_CONNECTION_NAME", "")),
			mustEnv("DB_NAME"),
			mustEnv("DB_USER"),
			mustEnv("DB_PASSWORD"),
			getEnv("DB_SSL_MODE", "disable"),
		)
	}

	return database.NewPostgreSQLRepository(
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		mustEnv("DB_NAME"),
		mustEnv("DB_USER"),
		mustEnv("DB_PASSWORD"),
		getEnv("DB_SSL_MODE", "disable"),
	)
}

func initStorageService() ports.StorageService {
	storageMode := strings.ToLower(getEnv("STORAGE_MODE", "gcs"))
	if storageMode == "local" {
		storagePath := getEnv("STORAGE_PATH", "./storage")
		staticBaseURL := getEnv("STATIC_BASE_URL", "")
		log.Printf("Storage mode: local (%s, baseURL: %s)", storagePath, staticBaseURL)
		return repositories.NewLocalStorageService(storagePath, staticBaseURL)
	}

	log.Printf("Storage mode: gcs")
	return repositories.NewGCSStorageService()
}

func parseAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return []string{"http://localhost:3000"}
	}
	return origins
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
