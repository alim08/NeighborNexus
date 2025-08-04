package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"neighborenexus/internal/middleware"
	"neighborenexus/internal/models"
	"neighborenexus/internal/services"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	websocketService *services.WebSocketService
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(websocketService *services.WebSocketService) *WebSocketHandler {
	return &WebSocketHandler{
		websocketService: websocketService,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create WebSocket client
	client := &services.WebSocketClient{
		ID:       uuid.New().String(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Service:  h.websocketService,
	}

	// Register client
	h.websocketService.register <- client

	// Start goroutines for reading and writing
	go client.readPump()
	go client.writePump()

	// Send welcome message
	welcomeMessage := models.WebSocketMessage{
		Type: "connected",
		Payload: map[string]interface{}{
			"user_id": userID,
			"message": "Connected to NeighborNexus",
		},
	}

	data, err := json.Marshal(welcomeMessage)
	if err == nil {
		client.Send <- data
	}
}

// upgrader is the WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
} 