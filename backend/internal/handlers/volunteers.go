package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"neighborenexus/internal/database"
	"neighborenexus/internal/middleware"
	"neighborenexus/internal/models"
	"neighborenexus/internal/services"
)

// VolunteerHandler handles volunteer-related requests
type VolunteerHandler struct {
	matchingService  *services.MatchingService
	websocketService *services.WebSocketService
	mongoClient      *database.MongoClient
}

// NewVolunteerHandler creates a new volunteer handler
func NewVolunteerHandler(matchingService *services.MatchingService, websocketService *services.WebSocketService, mongoClient *database.MongoClient) *VolunteerHandler {
	return &VolunteerHandler{
		matchingService:  matchingService,
		websocketService: websocketService,
		mongoClient:      mongoClient,
	}
}

// CreateProfile creates a volunteer profile
func (h *VolunteerHandler) CreateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateVolunteerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Convert user ID to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if volunteer profile already exists
	collection := h.mongoClient.GetCollection("volunteers")
	var existingVolunteer models.Volunteer
	err = collection.FindOne(c.Request.Context(), bson.M{"user_id": userObjectID}).Decode(&existingVolunteer)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Volunteer profile already exists"})
		return
	}

	// Create volunteer profile
	volunteer := models.Volunteer{
		ID:          primitive.NewObjectID(),
		UserID:      userObjectID,
		Skills:      req.Skills,
		Interests:   req.Interests,
		Description: req.Description,
		Availability: req.Availability,
		Location:    req.Location,
		Rating:      0.0,
		TaskCount:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert into database
	_, err = collection.InsertOne(c.Request.Context(), volunteer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create volunteer profile"})
		return
	}

	// Generate embedding for the volunteer
	if h.matchingService != nil {
		err = h.matchingService.UpdateVolunteerEmbedding(c.Request.Context(), &volunteer)
		if err != nil {
			// Log error but don't fail the request
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Profile created but embedding generation failed"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Volunteer profile created successfully",
		"volunteer": volunteer,
	})
}

// GetProfile retrieves the current user's volunteer profile
func (h *VolunteerHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	collection := h.mongoClient.GetCollection("volunteers")
	var volunteer models.Volunteer
	err = collection.FindOne(c.Request.Context(), bson.M{"user_id": userObjectID}).Decode(&volunteer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Volunteer profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve volunteer profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"volunteer": volunteer})
}

// UpdateProfile updates the current user's volunteer profile
func (h *VolunteerHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Skills      []string             `json:"skills,omitempty"`
		Interests   []string             `json:"interests,omitempty"`
		Description string               `json:"description,omitempty"`
		Availability []models.Availability `json:"availability,omitempty"`
		Location    models.Location      `json:"location,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Build update fields
	updates := bson.M{"updated_at": time.Now()}
	if len(req.Skills) > 0 {
		updates["skills"] = req.Skills
	}
	if len(req.Interests) > 0 {
		updates["interests"] = req.Interests
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if len(req.Availability) > 0 {
		updates["availability"] = req.Availability
	}
	if req.Location.Latitude != 0 || req.Location.Longitude != 0 {
		updates["location"] = req.Location
	}

	// Update in database
	collection := h.mongoClient.GetCollection("volunteers")
	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{"user_id": userObjectID},
		bson.M{"$set": updates},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update volunteer profile"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volunteer profile not found"})
		return
	}

	// Regenerate embedding if content changed
	if len(req.Skills) > 0 || len(req.Interests) > 0 || req.Description != "" {
		var volunteer models.Volunteer
		err = collection.FindOne(c.Request.Context(), bson.M{"user_id": userObjectID}).Decode(&volunteer)
		if err == nil && h.matchingService != nil {
			h.matchingService.UpdateVolunteerEmbedding(c.Request.Context(), &volunteer)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Volunteer profile updated successfully"})
}

// GetMatches retrieves matching needs for the current volunteer
func (h *VolunteerHandler) GetMatches(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get volunteer profile
	collection := h.mongoClient.GetCollection("volunteers")
	var volunteer models.Volunteer
	err = collection.FindOne(c.Request.Context(), bson.M{"user_id": userObjectID}).Decode(&volunteer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Volunteer profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve volunteer profile"})
		return
	}

	// Find matches for the volunteer
	var matches []models.Match
	if h.matchingService != nil {
		matches, err = h.matchingService.FindMatchesForVolunteer(c.Request.Context(), &volunteer, 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find matches"})
			return
		}
	}

	c.JSON(http.StatusOK, models.VolunteerResponse{
		Volunteer: volunteer,
		Matches:   matches,
	})
} 