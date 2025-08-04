package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string            `bson:"email" json:"email"`
	Password  string            `bson:"password" json:"-"`
	Name      string            `bson:"name" json:"name"`
	Phone     string            `bson:"phone,omitempty" json:"phone,omitempty"`
	Location  Location          `bson:"location" json:"location"`
	CreatedAt time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at" json:"updated_at"`
}

// Location represents a user's location (privacy-preserving)
type Location struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
	H3Index   string  `bson:"h3_index" json:"h3_index"` // Privacy-preserving location bucket
	Address   string  `bson:"address,omitempty" json:"address,omitempty"`
}

// Need represents a user's request for help
type Need struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Title       string            `bson:"title" json:"title"`
	Description string            `bson:"description" json:"description"`
	Category    string            `bson:"category" json:"category"`
	Urgency     string            `bson:"urgency" json:"urgency"` // low, medium, high
	Duration    int               `bson:"duration" json:"duration"` // estimated minutes
	Location    Location          `bson:"location" json:"location"`
	Status      string            `bson:"status" json:"status"` // requested, matched, in_progress, completed, cancelled
	Embedding   []float32         `bson:"embedding,omitempty" json:"-"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
	ExpiresAt   *time.Time        `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
}

// Volunteer represents a volunteer's profile
type Volunteer struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Skills      []string          `bson:"skills" json:"skills"`
	Interests   []string          `bson:"interests" json:"interests"`
	Description string            `bson:"description" json:"description"`
	Availability []Availability    `bson:"availability" json:"availability"`
	Location    Location          `bson:"location" json:"location"`
	Embedding   []float32         `bson:"embedding,omitempty" json:"-"`
	Rating      float64           `bson:"rating" json:"rating"`
	TaskCount   int               `bson:"task_count" json:"task_count"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
}

// Availability represents when a volunteer is available
type Availability struct {
	DayOfWeek int    `bson:"day_of_week" json:"day_of_week"` // 0=Sunday, 1=Monday, etc.
	StartTime string `bson:"start_time" json:"start_time"`    // "09:00"
	EndTime   string `bson:"end_time" json:"end_time"`        // "17:00"
}

// Task represents a matched need that is being worked on
type Task struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	NeedID       primitive.ObjectID `bson:"need_id" json:"need_id"`
	VolunteerID  primitive.ObjectID `bson:"volunteer_id" json:"volunteer_id"`
	Status       string            `bson:"status" json:"status"` // accepted, in_progress, completed, cancelled
	ScheduledAt  *time.Time        `bson:"scheduled_at,omitempty" json:"scheduled_at,omitempty"`
	CompletedAt  *time.Time        `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	Notes        string            `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `bson:"updated_at" json:"updated_at"`
}

// Feedback represents feedback given after task completion
type Feedback struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID       primitive.ObjectID `bson:"task_id" json:"task_id"`
	FromUserID   primitive.ObjectID `bson:"from_user_id" json:"from_user_id"`
	ToUserID     primitive.ObjectID `bson:"to_user_id" json:"to_user_id"`
	Rating       int               `bson:"rating" json:"rating"` // 1-5 stars
	Comment      string            `bson:"comment,omitempty" json:"comment,omitempty"`
	CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
}

// Match represents a potential match between a need and volunteer
type Match struct {
	NeedID      primitive.ObjectID `bson:"need_id" json:"need_id"`
	VolunteerID primitive.ObjectID `bson:"volunteer_id" json:"volunteer_id"`
	Score       float64            `bson:"score" json:"score"` // similarity score
	Distance    float64            `bson:"distance" json:"distance"` // distance in meters
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// WebSocketMessage represents a message sent via WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	UserID  string      `json:"user_id,omitempty"`
}

// API Response structures
type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type NeedResponse struct {
	Need    Need     `json:"need"`
	Matches []Match  `json:"matches,omitempty"`
}

type VolunteerResponse struct {
	Volunteer Volunteer `json:"volunteer"`
	Matches   []Match   `json:"matches,omitempty"`
}

// Request structures
type RegisterRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=6"`
	Name     string   `json:"name" binding:"required"`
	Phone    string   `json:"phone,omitempty"`
	Location Location `json:"location" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateNeedRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Category    string   `json:"category" binding:"required"`
	Urgency     string   `json:"urgency" binding:"required"`
	Duration    int      `json:"duration" binding:"required"`
	Location    Location `json:"location" binding:"required"`
}

type CreateVolunteerRequest struct {
	Skills      []string       `json:"skills" binding:"required"`
	Interests   []string       `json:"interests"`
	Description string         `json:"description" binding:"required"`
	Availability []Availability `json:"availability"`
	Location    Location       `json:"location" binding:"required"`
}

type UpdateTaskStatusRequest struct {
	Status      string     `json:"status" binding:"required"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

type FeedbackRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment,omitempty"`
} 