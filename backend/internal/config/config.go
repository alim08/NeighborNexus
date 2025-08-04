package config

import (
	"os"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	Port string

	// Database settings
	MongoURI      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// JWT settings
	JWTSecret string

	// OpenAI settings
	OpenAIKey string

	// Pinecone settings
	PineconeAPIKey string
	PineconeIndex  string

	// Environment
	Environment string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        0, // Default Redis database
		JWTSecret:      getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		OpenAIKey:      getEnv("OPENAI_API_KEY", ""),
		PineconeAPIKey: getEnv("PINECONE_API_KEY", ""),
		PineconeIndex:  getEnv("PINECONE_INDEX", "neighborenexus"),
		Environment:    getEnv("ENVIRONMENT", "development"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
} 