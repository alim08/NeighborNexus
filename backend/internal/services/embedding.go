package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingService handles OpenAI embeddings for semantic matching
type EmbeddingService struct {
	client *openai.Client
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(apiKey string) *EmbeddingService {
	if apiKey == "" {
		log.Println("Warning: OpenAI API key not provided, embedding service will not work")
		return &EmbeddingService{
			client: nil,
		}
	}

	return &EmbeddingService{
		client: openai.NewClient(apiKey),
	}
}

// GenerateEmbedding creates an embedding for the given text
func (e *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if e.client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	// Clean and prepare text
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Truncate text if too long (OpenAI has limits)
	if len(text) > 8000 {
		text = text[:8000]
	}

	resp, err := e.client.CreateEmbeddings(
		ctx,
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.AdaEmbeddingV2,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

// GenerateNeedEmbedding creates an embedding for a need description
func (e *EmbeddingService) GenerateNeedEmbedding(ctx context.Context, title, description, category string) ([]float32, error) {
	// Combine title, description, and category for better semantic matching
	text := fmt.Sprintf("Title: %s\nDescription: %s\nCategory: %s", title, description, category)
	return e.GenerateEmbedding(ctx, text)
}

// GenerateVolunteerEmbedding creates an embedding for a volunteer profile
func (e *EmbeddingService) GenerateVolunteerEmbedding(ctx context.Context, skills, interests, description []string) ([]float32, error) {
	// Combine skills, interests, and description for better semantic matching
	text := fmt.Sprintf("Skills: %s\nInterests: %s\nDescription: %s",
		strings.Join(skills, ", "),
		strings.Join(interests, ", "),
		strings.Join(description, " "))
	return e.GenerateEmbedding(ctx, text)
}

// BatchGenerateEmbeddings creates embeddings for multiple texts
func (e *EmbeddingService) BatchGenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if e.client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	// Clean and truncate texts
	cleanedTexts := make([]string, len(texts))
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if len(text) > 8000 {
			text = text[:8000]
		}
		cleanedTexts[i] = text
	}

	resp, err := e.client.CreateEmbeddings(
		ctx,
		openai.EmbeddingRequest{
			Input: cleanedTexts,
			Model: openai.AdaEmbeddingV2,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate batch embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// CalculateSimilarity calculates cosine similarity between two embeddings
func (e *EmbeddingService) CalculateSimilarity(embedding1, embedding2 []float32) (float64, error) {
	if len(embedding1) != len(embedding2) {
		return 0, fmt.Errorf("embedding dimensions do not match")
	}

	if len(embedding1) == 0 {
		return 0, fmt.Errorf("embeddings cannot be empty")
	}

	// Calculate dot product
	var dotProduct float64
	var norm1 float64
	var norm2 float64

	for i := 0; i < len(embedding1); i++ {
		dotProduct += float64(embedding1[i] * embedding2[i])
		norm1 += float64(embedding1[i] * embedding1[i])
		norm2 += float64(embedding2[i] * embedding2[i])
	}

	// Calculate cosine similarity
	norm1 = sqrt(norm1)
	norm2 = sqrt(norm2)

	if norm1 == 0 || norm2 == 0 {
		return 0, nil
	}

	return dotProduct / (norm1 * norm2), nil
}

// sqrt calculates the square root (simplified version)
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	
	// Newton's method for square root
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// IsAvailable checks if the embedding service is available
func (e *EmbeddingService) IsAvailable() bool {
	return e.client != nil
}

// GetEmbeddingInfo returns information about the embedding service
func (e *EmbeddingService) GetEmbeddingInfo() map[string]interface{} {
	return map[string]interface{}{
		"available": e.IsAvailable(),
		"model":     "text-embedding-ada-002",
		"dimensions": 1536,
	}
} 