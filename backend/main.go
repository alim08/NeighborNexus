package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"neighborenexus/internal/config"
	"neighborenexus/internal/database"
	"neighborenexus/internal/handlers"
	"neighborenexus/internal/middleware"
	"neighborenexus/internal/services"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize database connections
	mongoClient, err := database.NewMongoClient(cfg.MongoURI)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(nil)

	redisClient := database.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redisClient.Close()

	// Initialize services
	authService := services.NewAuthService(mongoClient, cfg.JWTSecret)
	embeddingService := services.NewEmbeddingService(cfg.OpenAIKey)
	matchingService := services.NewMatchingService(embeddingService, mongoClient, cfg.PineconeAPIKey, cfg.PineconeIndex)
	websocketService := services.NewWebSocketService()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	needHandler := handlers.NewNeedHandler(matchingService, websocketService)
	volunteerHandler := handlers.NewVolunteerHandler(matchingService, websocketService)
	websocketHandler := handlers.NewWebSocketHandler(websocketService)

	// Setup Gin router
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "neighborenexus"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// User profile
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)

			// Needs
			needs := protected.Group("/needs")
			{
				needs.POST("/", needHandler.CreateNeed)
				needs.GET("/", needHandler.GetNeeds)
				needs.GET("/:id", needHandler.GetNeed)
				needs.PUT("/:id", needHandler.UpdateNeed)
				needs.DELETE("/:id", needHandler.DeleteNeed)
				needs.POST("/:id/accept", needHandler.AcceptNeed)
			}

			// Volunteers
			volunteers := protected.Group("/volunteers")
			{
				volunteers.POST("/profile", volunteerHandler.CreateProfile)
				volunteers.GET("/profile", volunteerHandler.GetProfile)
				volunteers.PUT("/profile", volunteerHandler.UpdateProfile)
				volunteers.GET("/matches", volunteerHandler.GetMatches)
			}

			// Tasks
			tasks := protected.Group("/tasks")
			{
				tasks.GET("/", needHandler.GetTasks)
				tasks.GET("/:id", needHandler.GetTask)
				tasks.PUT("/:id/status", needHandler.UpdateTaskStatus)
				tasks.POST("/:id/feedback", needHandler.SubmitFeedback)
			}
		}

		// WebSocket endpoint
		api.GET("/ws", middleware.AuthMiddleware(authService), websocketHandler.HandleWebSocket)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting NeighborNexus server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
} 