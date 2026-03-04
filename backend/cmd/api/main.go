package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

	// Auto-migrate: This creates or updates tables based on domain models.
	// For production, use SQL migration files in /migrations instead.
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// --- Dependency Injection (Composition Root) ---
	// This is the ONLY place where concrete implementations are wired together.
	// Think of it as the "startup configuration" of our Clean Architecture tree.

	userRepo := repositories.NewPostgresUserRepository(db)

	jwtSvc := repositories.NewJWTService(
		mustEnv("JWT_SECRET"),
		mustEnv("REFRESH_TOKEN_SECRET"),
	)

	googleVerifier := repositories.NewGoogleVerifier(mustEnv("GOOGLE_CLIENT_ID"))

	authUseCase := usecases.NewAuthUseCase(userRepo, jwtSvc, googleVerifier)

	// --- HTTP Router setup ---
	router := gin.Default()

	// Health check endpoint — used by Cloud Run and load balancers.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Public routes (no auth required)
	authHandler := httphandlers.NewAuthHandler(authUseCase)
	authHandler.RegisterRoutes(v1)

	// Protected routes (JWT required) — example structure for future modules
	// protected := v1.Group("/")
	// protected.Use(httphandlers.AuthMiddleware(jwtSvc))
	// courseHandler.RegisterRoutes(protected)

	port := getEnv("PORT", "8080")
	log.Printf("Klyra Backend starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
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
