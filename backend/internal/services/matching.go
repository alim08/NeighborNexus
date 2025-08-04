package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/uber/h3-go/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"neighborenexus/internal/database"
	"neighborenexus/internal/models"
)

// MatchingService handles semantic matching between needs and volunteers
type MatchingService struct {
	embeddingService *EmbeddingService
	mongoClient      *database.MongoClient
	pineconeAPIKey   string
	pineconeIndex    string
}

// NewMatchingService creates a new matching service
func NewMatchingService(embeddingService *EmbeddingService, mongoClient *database.MongoClient, pineconeAPIKey, pineconeIndex string) *MatchingService {
	return &MatchingService{
		embeddingService: embeddingService,
		mongoClient:      mongoClient,
		pineconeAPIKey:   pineconeAPIKey,
		pineconeIndex:    pineconeIndex,
	}
}

// FindMatchesForNeed finds matching volunteers for a specific need
func (m *MatchingService) FindMatchesForNeed(ctx context.Context, need *models.Need, limit int) ([]models.Match, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get all active volunteers
	volunteers, err := m.getActiveVolunteers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get volunteers: %w", err)
	}

	var matches []models.Match

	// Calculate similarity scores for each volunteer
	for _, volunteer := range volunteers {
		// Skip if volunteer has no embedding
		if len(volunteer.Embedding) == 0 {
			continue
		}

		// Calculate semantic similarity
		similarity, err := m.embeddingService.CalculateSimilarity(need.Embedding, volunteer.Embedding)
		if err != nil {
			continue // Skip this volunteer if similarity calculation fails
		}

		// Calculate distance
		distance := m.calculateDistance(need.Location, volunteer.Location)

		// Apply distance penalty (closer is better)
		distanceScore := m.calculateDistanceScore(distance)

		// Combine similarity and distance scores
		combinedScore := similarity * distanceScore

		// Only include matches above threshold
		if combinedScore > 0.3 {
			matches = append(matches, models.Match{
				NeedID:      need.ID,
				VolunteerID: volunteer.ID,
				Score:       combinedScore,
				Distance:    distance,
				CreatedAt:   time.Now(),
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Return top matches
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// FindMatchesForVolunteer finds matching needs for a specific volunteer
func (m *MatchingService) FindMatchesForVolunteer(ctx context.Context, volunteer *models.Volunteer, limit int) ([]models.Match, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get all active needs
	needs, err := m.getActiveNeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get needs: %w", err)
	}

	var matches []models.Match

	// Calculate similarity scores for each need
	for _, need := range needs {
		// Skip if need has no embedding
		if len(need.Embedding) == 0 {
			continue
		}

		// Calculate semantic similarity
		similarity, err := m.embeddingService.CalculateSimilarity(volunteer.Embedding, need.Embedding)
		if err != nil {
			continue // Skip this need if similarity calculation fails
		}

		// Calculate distance
		distance := m.calculateDistance(need.Location, volunteer.Location)

		// Apply distance penalty (closer is better)
		distanceScore := m.calculateDistanceScore(distance)

		// Combine similarity and distance scores
		combinedScore := similarity * distanceScore

		// Only include matches above threshold
		if combinedScore > 0.3 {
			matches = append(matches, models.Match{
				NeedID:      need.ID,
				VolunteerID: volunteer.ID,
				Score:       combinedScore,
				Distance:    distance,
				CreatedAt:   time.Now(),
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Return top matches
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// getActiveVolunteers retrieves all active volunteers
func (m *MatchingService) getActiveVolunteers(ctx context.Context) ([]models.Volunteer, error) {
	collection := m.mongoClient.GetCollection("volunteers")
	
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var volunteers []models.Volunteer
	if err = cursor.All(ctx, &volunteers); err != nil {
		return nil, err
	}

	return volunteers, nil
}

// getActiveNeeds retrieves all active needs
func (m *MatchingService) getActiveNeeds(ctx context.Context) ([]models.Need, error) {
	collection := m.mongoClient.GetCollection("needs")
	
	// Only get needs that are still open
	filter := bson.M{
		"status": bson.M{"$in": []string{"requested", "matched"}},
		"$or": []bson.M{
			{"expires_at": bson.M{"$exists": false}},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var needs []models.Need
	if err = cursor.All(ctx, &needs); err != nil {
		return nil, err
	}

	return needs, nil
}

// calculateDistance calculates the distance between two locations in meters
func (m *MatchingService) calculateDistance(loc1, loc2 models.Location) float64 {
	// Convert to radians
	lat1 := loc1.Latitude * math.Pi / 180
	lon1 := loc1.Longitude * math.Pi / 180
	lat2 := loc2.Latitude * math.Pi / 180
	lon2 := loc2.Longitude * math.Pi / 180

	// Haversine formula
	dlat := lat2 - lat1
	dlon := lon2 - lon1
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Earth's radius in meters
	earthRadius := 6371000.0

	return earthRadius * c
}

// calculateDistanceScore calculates a score based on distance (closer is better)
func (m *MatchingService) calculateDistanceScore(distance float64) float64 {
	// Convert distance to kilometers
	distanceKm := distance / 1000

	// Use exponential decay: score = e^(-distance/10)
	// This gives a score of 1.0 for 0km, 0.37 for 10km, 0.14 for 20km, etc.
	return math.Exp(-distanceKm / 10.0)
}

// GenerateH3Index generates an H3 index for privacy-preserving location matching
func (m *MatchingService) GenerateH3Index(lat, lng float64, resolution int) string {
	// Create H3 index at the specified resolution
	index := h3.LatLngToCell(h3.LatLng{
		Lat: lat,
		Lng: lng,
	}, h3.Res(resolution))

	return index.String()
}

// GetNearbyH3Indices gets nearby H3 indices for proximity filtering
func (m *MatchingService) GetNearbyH3Indices(h3Index string, radiusKm float64) ([]string, error) {
	index, err := h3.CellFromString(h3Index)
	if err != nil {
		return nil, err
	}

	// Get indices within the specified radius
	indices := h3.GridDisk(index, int(radiusKm))
	
	result := make([]string, len(indices))
	for i, idx := range indices {
		result[i] = idx.String()
	}

	return result, nil
}

// UpdateNeedEmbedding updates the embedding for a need
func (m *MatchingService) UpdateNeedEmbedding(ctx context.Context, need *models.Need) error {
	if !m.embeddingService.IsAvailable() {
		return fmt.Errorf("embedding service not available")
	}

	embedding, err := m.embeddingService.GenerateNeedEmbedding(
		ctx,
		need.Title,
		need.Description,
		need.Category,
	)
	if err != nil {
		return fmt.Errorf("failed to generate need embedding: %w", err)
	}

	// Update the need with the new embedding
	collection := m.mongoClient.GetCollection("needs")
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": need.ID},
		bson.M{"$set": bson.M{
			"embedding":   embedding,
			"updated_at":  time.Now(),
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update need embedding: %w", err)
	}

	need.Embedding = embedding
	return nil
}

// UpdateVolunteerEmbedding updates the embedding for a volunteer
func (m *MatchingService) UpdateVolunteerEmbedding(ctx context.Context, volunteer *models.Volunteer) error {
	if !m.embeddingService.IsAvailable() {
		return fmt.Errorf("embedding service not available")
	}

	embedding, err := m.embeddingService.GenerateVolunteerEmbedding(
		ctx,
		volunteer.Skills,
		volunteer.Interests,
		[]string{volunteer.Description},
	)
	if err != nil {
		return fmt.Errorf("failed to generate volunteer embedding: %w", err)
	}

	// Update the volunteer with the new embedding
	collection := m.mongoClient.GetCollection("volunteers")
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": volunteer.ID},
		bson.M{"$set": bson.M{
			"embedding":   embedding,
			"updated_at":  time.Now(),
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update volunteer embedding: %w", err)
	}

	volunteer.Embedding = embedding
	return nil
} 