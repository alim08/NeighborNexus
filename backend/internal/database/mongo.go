package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient wraps the MongoDB client
type MongoClient struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// NewMongoClient creates a new MongoDB client
func NewMongoClient(uri string) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := client.Database("neighborenexus")

	// Create indexes
	if err := createIndexes(ctx, db); err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}

	return &MongoClient{
		Client: client,
		DB:     db,
	}, nil
}

// createIndexes creates necessary indexes for the application
func createIndexes(ctx context.Context, db *mongo.Database) error {
	// Users collection indexes
	usersCollection := db.Collection("users")
	_, err := usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"email": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Needs collection indexes
	needsCollection := db.Collection("needs")
	_, err = needsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"status": 1,
		},
	})
	if err != nil {
		return err
	}

	_, err = needsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"user_id": 1,
		},
	})
	if err != nil {
		return err
	}

	_, err = needsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"created_at": -1,
		},
	})
	if err != nil {
		return err
	}

	// Volunteers collection indexes
	volunteersCollection := db.Collection("volunteers")
	_, err = volunteersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"user_id": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Tasks collection indexes
	tasksCollection := db.Collection("tasks")
	_, err = tasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"need_id": 1,
		},
	})
	if err != nil {
		return err
	}

	_, err = tasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"volunteer_id": 1,
		},
	})
	if err != nil {
		return err
	}

	_, err = tasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"status": 1,
		},
	})
	if err != nil {
		return err
	}

	// Feedback collection indexes
	feedbackCollection := db.Collection("feedback")
	_, err = feedbackCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"task_id": 1,
		},
	})
	if err != nil {
		return err
	}

	_, err = feedbackCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"to_user_id": 1,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// GetCollection returns a MongoDB collection
func (m *MongoClient) GetCollection(name string) *mongo.Collection {
	return m.DB.Collection(name)
}

// Close closes the MongoDB connection
func (m *MongoClient) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
} 