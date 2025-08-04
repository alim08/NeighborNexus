package services

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"neighborenexus/internal/database"
	"neighborenexus/internal/models"
)

// AuthService handles authentication and user management
type AuthService struct {
	mongoClient *database.MongoClient
	jwtSecret   string
}

// NewAuthService creates a new authentication service
func NewAuthService(mongoClient *database.MongoClient, jwtSecret string) *AuthService {
	return &AuthService{
		mongoClient: mongoClient,
		jwtSecret:   jwtSecret,
	}
}

// Register creates a new user account
func (a *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	// Check if user already exists
	collection := a.mongoClient.GetCollection("users")
	var existingUser models.User
	err := collection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		return nil, errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := models.User{
		ID:        primitive.NewObjectID(),
		Email:     req.Email,
		Password:  string(hashedPassword),
		Name:      req.Name,
		Phone:     req.Phone,
		Location:  req.Location,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert user into database
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	// Clear password from response
	user.Password = ""
	return &user, nil
}

// Login authenticates a user and returns JWT tokens
func (a *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	// Find user by email
	collection := a.mongoClient.GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate JWT tokens
	accessToken, err := a.generateAccessToken(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.generateRefreshToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	// Clear password from response
	user.Password = ""

	return &models.AuthResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// RefreshToken generates a new access token using a refresh token
func (a *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	// Extract user ID from token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid user ID in token")
	}

	// Get user from database
	collection := a.mongoClient.GetCollection("users")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var user models.User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate new tokens
	accessToken, err := a.generateAccessToken(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := a.generateRefreshToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	// Clear password from response
	user.Password = ""

	return &models.AuthResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

// GetUserByID retrieves a user by ID
func (a *AuthService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	collection := a.mongoClient.GetCollection("users")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var user models.User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	// Clear password from response
	user.Password = ""
	return &user, nil
}

// UpdateUser updates a user's profile
func (a *AuthService) UpdateUser(ctx context.Context, userID string, updates bson.M) (*models.User, error) {
	collection := a.mongoClient.GetCollection("users")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Add updated_at timestamp
	updates["updated_at"] = time.Now()

	// Update user
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("user not found")
	}

	// Return updated user
	return a.GetUserByID(ctx, userID)
}

// generateAccessToken creates a JWT access token
func (a *AuthService) generateAccessToken(userID, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"type":    "access",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// generateRefreshToken creates a JWT refresh token
func (a *AuthService) generateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// ValidateToken validates a JWT token and returns the user ID
func (a *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.New("invalid user ID in token")
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return "", errors.New("invalid token type")
	}

	return userID, nil
} 