package services

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"neighborenexus/internal/models"
)

// WebSocketService handles real-time WebSocket connections
type WebSocketService struct {
	clients    map[string]*WebSocketClient
	broadcast  chan models.WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mutex      sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Service  *WebSocketService
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService() *WebSocketService {
	return &WebSocketService{
		clients:    make(map[string]*WebSocketClient),
		broadcast:  make(chan models.WebSocketMessage),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Start starts the WebSocket service
func (ws *WebSocketService) Start() {
	for {
		select {
		case client := <-ws.register:
			ws.mutex.Lock()
			ws.clients[client.ID] = client
			ws.mutex.Unlock()
			log.Printf("WebSocket client registered: %s (User: %s)", client.ID, client.UserID)

		case client := <-ws.unregister:
			ws.mutex.Lock()
			if _, ok := ws.clients[client.ID]; ok {
				delete(ws.clients, client.ID)
				close(client.Send)
			}
			ws.mutex.Unlock()
			log.Printf("WebSocket client unregistered: %s (User: %s)", client.ID, client.UserID)

		case message := <-ws.broadcast:
			ws.broadcastMessage(message)
		}
	}
}

// broadcastMessage sends a message to all connected clients
func (ws *WebSocketService) broadcastMessage(message models.WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	for _, client := range ws.clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(ws.clients, client.ID)
		}
	}
}

// SendToUser sends a message to a specific user
func (ws *WebSocketService) SendToUser(userID string, message models.WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	for _, client := range ws.clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(ws.clients, client.ID)
			}
		}
	}
}

// SendToMultipleUsers sends a message to multiple users
func (ws *WebSocketService) SendToMultipleUsers(userIDs []string, message models.WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	userIDSet := make(map[string]bool)
	for _, id := range userIDs {
		userIDSet[id] = true
	}

	for _, client := range ws.clients {
		if userIDSet[client.UserID] {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(ws.clients, client.ID)
			}
		}
	}
}

// NotifyNewNeed notifies relevant volunteers about a new need
func (ws *WebSocketService) NotifyNewNeed(need models.Need, volunteerIDs []string) {
	message := models.WebSocketMessage{
		Type: "new_need",
		Payload: map[string]interface{}{
			"need_id": need.ID.Hex(),
			"title":   need.Title,
			"urgency": need.Urgency,
		},
	}

	ws.SendToMultipleUsers(volunteerIDs, message)
}

// NotifyNeedAccepted notifies the need creator that their need was accepted
func (ws *WebSocketService) NotifyNeedAccepted(needID, volunteerID string, volunteerName string) {
	message := models.WebSocketMessage{
		Type: "need_accepted",
		Payload: map[string]interface{}{
			"need_id":       needID,
			"volunteer_id":  volunteerID,
			"volunteer_name": volunteerName,
		},
	}

	// Send to need creator
	ws.SendToUser(needID, message)
}

// NotifyTaskStatusUpdate notifies users about task status changes
func (ws *WebSocketService) NotifyTaskStatusUpdate(task models.Task, userIDs []string) {
	message := models.WebSocketMessage{
		Type: "task_status_update",
		Payload: map[string]interface{}{
			"task_id": task.ID.Hex(),
			"status":  task.Status,
		},
	}

	ws.SendToMultipleUsers(userIDs, message)
}

// NotifyNewMatch notifies users about new matches
func (ws *WebSocketService) NotifyNewMatch(match models.Match, userIDs []string) {
	message := models.WebSocketMessage{
		Type: "new_match",
		Payload: map[string]interface{}{
			"match_id": match.NeedID.Hex(),
			"score":    match.Score,
			"distance": match.Distance,
		},
	}

	ws.SendToMultipleUsers(userIDs, message)
}

// GetConnectedUsers returns a list of connected user IDs
func (ws *WebSocketService) GetConnectedUsers() []string {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	userIDs := make([]string, 0, len(ws.clients))
	for _, client := range ws.clients {
		userIDs = append(userIDs, client.UserID)
	}

	return userIDs
}

// IsUserConnected checks if a user is currently connected
func (ws *WebSocketService) IsUserConnected(userID string) bool {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	for _, client := range ws.clients {
		if client.UserID == userID {
			return true
		}
	}

	return false
}

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Service.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages if needed
		log.Printf("Received message from client %s: %s", c.ID, string(message))
	}
}

// writePump writes messages to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Upgrader for WebSocket connections
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
} 