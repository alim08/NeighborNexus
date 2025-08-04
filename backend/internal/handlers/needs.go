package handlers

import (
	"context"
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

// NeedHandler handles need-related requests
type NeedHandler struct {
	matchingService   *services.MatchingService
	websocketService  *services.WebSocketService
	mongoClient       *database.MongoClient
}

// NewNeedHandler creates a new need handler
func NewNeedHandler(matchingService *services.MatchingService, websocketService *services.WebSocketService, mongoClient *database.MongoClient) *NeedHandler {
	return &NeedHandler{
		matchingService:  matchingService,
		websocketService: websocketService,
		mongoClient:      mongoClient,
	}
}

// CreateNeed creates a new need
func (h *NeedHandler) CreateNeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateNeedRequest
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

	// Create need
	need := models.Need{
		ID:          primitive.NewObjectID(),
		UserID:      userObjectID,
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Urgency:     req.Urgency,
		Duration:    req.Duration,
		Location:    req.Location,
		Status:      "requested",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set expiration (default 7 days)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	need.ExpiresAt = &expiresAt

	// Insert into database
	collection := h.mongoClient.GetCollection("needs")
	_, err = collection.InsertOne(c.Request.Context(), need)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create need"})
		return
	}

	// Generate embedding for the need
	if h.matchingService != nil {
		err = h.matchingService.UpdateNeedEmbedding(c.Request.Context(), &need)
		if err != nil {
			// Log error but don't fail the request
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Need created but embedding generation failed"})
			return
		}
	}

	// Find matches for the need
	var matches []models.Match
	if h.matchingService != nil {
		matches, err = h.matchingService.FindMatchesForNeed(c.Request.Context(), &need, 5)
		if err != nil {
			// Log error but don't fail the request
		}
	}

	// Notify relevant volunteers via WebSocket
	if h.websocketService != nil && len(matches) > 0 {
		volunteerIDs := make([]string, len(matches))
		for i, match := range matches {
			volunteerIDs[i] = match.VolunteerID.Hex()
		}
		h.websocketService.NotifyNewNeed(need, volunteerIDs)
	}

	c.JSON(http.StatusCreated, models.NeedResponse{
		Need:    need,
		Matches: matches,
	})
}

// GetNeeds retrieves needs with optional filtering
func (h *NeedHandler) GetNeeds(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse query parameters
	status := c.Query("status")
	category := c.Query("category")
	limit := 20 // Default limit

	// Build filter
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	if category != "" {
		filter["category"] = category
	}

	// Add expiration filter
	filter["$or"] = []bson.M{
		{"expires_at": bson.M{"$exists": false}},
		{"expires_at": bson.M{"$gt": time.Now()}},
	}

	// Query database
	collection := h.mongoClient.GetCollection("needs")
	opts := mongo.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit))
	
	cursor, err := collection.Find(c.Request.Context(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve needs"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var needs []models.Need
	if err = cursor.All(c.Request.Context(), &needs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode needs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"needs": needs})
}

// GetNeed retrieves a specific need
func (h *NeedHandler) GetNeed(c *gin.Context) {
	needID := c.Param("id")
	if needID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need ID required"})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(needID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid need ID"})
		return
	}

	collection := h.mongoClient.GetCollection("needs")
	var need models.Need
	err = collection.FindOne(c.Request.Context(), bson.M{"_id": objectID}).Decode(&need)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Need not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve need"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"need": need})
}

// UpdateNeed updates a need
func (h *NeedHandler) UpdateNeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	needID := c.Param("id")
	if needID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need ID required"})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(needID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid need ID"})
		return
	}

	var req struct {
		Title       string            `json:"title,omitempty"`
		Description string            `json:"description,omitempty"`
		Category    string            `json:"category,omitempty"`
		Urgency     string            `json:"urgency,omitempty"`
		Duration    int               `json:"duration,omitempty"`
		Location    models.Location   `json:"location,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Build update fields
	updates := bson.M{"updated_at": time.Now()}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Urgency != "" {
		updates["urgency"] = req.Urgency
	}
	if req.Duration > 0 {
		updates["duration"] = req.Duration
	}
	if req.Location.Latitude != 0 || req.Location.Longitude != 0 {
		updates["location"] = req.Location
	}

	// Update in database
	collection := h.mongoClient.GetCollection("needs")
	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{"_id": objectID, "user_id": userID}, // Only allow owner to update
		bson.M{"$set": updates},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update need"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Need not found or not owned by user"})
		return
	}

	// Regenerate embedding if content changed
	if req.Title != "" || req.Description != "" || req.Category != "" {
		var need models.Need
		err = collection.FindOne(c.Request.Context(), bson.M{"_id": objectID}).Decode(&need)
		if err == nil && h.matchingService != nil {
			h.matchingService.UpdateNeedEmbedding(c.Request.Context(), &need)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Need updated successfully"})
}

// DeleteNeed deletes a need
func (h *NeedHandler) DeleteNeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	needID := c.Param("id")
	if needID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need ID required"})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(needID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid need ID"})
		return
	}

	collection := h.mongoClient.GetCollection("needs")
	result, err := collection.DeleteOne(
		c.Request.Context(),
		bson.M{"_id": objectID, "user_id": userID}, // Only allow owner to delete
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete need"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Need not found or not owned by user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Need deleted successfully"})
}

// AcceptNeed accepts a need (creates a task)
func (h *NeedHandler) AcceptNeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	needID := c.Param("id")
	if needID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need ID required"})
		return
	}

	needObjectID, err := primitive.ObjectIDFromHex(needID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid need ID"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if need exists and is available
	needsCollection := h.mongoClient.GetCollection("needs")
	var need models.Need
	err = needsCollection.FindOne(c.Request.Context(), bson.M{"_id": needObjectID, "status": "requested"}).Decode(&need)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Need not found or already accepted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve need"})
		return
	}

	// Check if user is not the need creator
	if need.UserID == userObjectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot accept your own need"})
		return
	}

	// Create task
	task := models.Task{
		ID:          primitive.NewObjectID(),
		NeedID:      needObjectID,
		VolunteerID: userObjectID,
		Status:      "accepted",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tasksCollection := h.mongoClient.GetCollection("tasks")
	_, err = tasksCollection.InsertOne(c.Request.Context(), task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Update need status
	_, err = needsCollection.UpdateOne(
		c.Request.Context(),
		bson.M{"_id": needObjectID},
		bson.M{"$set": bson.M{"status": "matched", "updated_at": time.Now()}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update need status"})
		return
	}

	// Notify need creator via WebSocket
	if h.websocketService != nil {
		needCreatorID := need.UserID.Hex()
		h.websocketService.NotifyNeedAccepted(needID, userID, "Volunteer") // You'd get the actual volunteer name
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Need accepted successfully",
		"task":    task,
	})
}

// GetTasks retrieves tasks for the current user
func (h *NeedHandler) GetTasks(c *gin.Context) {
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

	// Get tasks where user is either the need creator or the volunteer
	collection := h.mongoClient.GetCollection("tasks")
	filter := bson.M{
		"$or": []bson.M{
			{"volunteer_id": userObjectID},
		},
	}

	cursor, err := collection.Find(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var tasks []models.Task
	if err = cursor.All(c.Request.Context(), &tasks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// GetTask retrieves a specific task
func (h *NeedHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID required"})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	collection := h.mongoClient.GetCollection("tasks")
	var task models.Task
	err = collection.FindOne(c.Request.Context(), bson.M{"_id": objectID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

// UpdateTaskStatus updates a task's status
func (h *NeedHandler) UpdateTaskStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID required"})
		return
	}

	var req models.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Build update fields
	updates := bson.M{
		"status":     req.Status,
		"updated_at": time.Now(),
	}
	if req.ScheduledAt != nil {
		updates["scheduled_at"] = req.ScheduledAt
	}
	if req.Notes != "" {
		updates["notes"] = req.Notes
	}

	// Update task
	collection := h.mongoClient.GetCollection("tasks")
	result, err := collection.UpdateOne(
		c.Request.Context(),
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task status updated successfully"})
}

// SubmitFeedback submits feedback for a completed task
func (h *NeedHandler) SubmitFeedback(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID required"})
		return
	}

	var req models.FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get task to determine who to give feedback to
	collection := h.mongoClient.GetCollection("tasks")
	var task models.Task
	err = collection.FindOne(c.Request.Context(), bson.M{"_id": objectID}).Decode(&task)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Determine who is giving feedback to whom
	var fromUserID, toUserID primitive.ObjectID
	if task.VolunteerID == userObjectID {
		// Volunteer is giving feedback to need creator
		fromUserID = userObjectID
		toUserID = task.NeedID // This should be the need creator's ID, but we need to get it from the need
		
		// Get the need to find the creator
		needsCollection := h.mongoClient.GetCollection("needs")
		var need models.Need
		err = needsCollection.FindOne(c.Request.Context(), bson.M{"_id": task.NeedID}).Decode(&need)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get need details"})
			return
		}
		toUserID = need.UserID
	} else {
		// Need creator is giving feedback to volunteer
		fromUserID = userObjectID
		toUserID = task.VolunteerID
	}

	// Create feedback
	feedback := models.Feedback{
		ID:         primitive.NewObjectID(),
		TaskID:     objectID,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
	}

	feedbackCollection := h.mongoClient.GetCollection("feedback")
	_, err = feedbackCollection.InsertOne(c.Request.Context(), feedback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit feedback"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Feedback submitted successfully",
		"feedback": feedback,
	})
} 