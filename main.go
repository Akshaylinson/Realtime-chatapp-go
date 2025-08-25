package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Message represents a chat message
type Message struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// Client represents a connected user
type Client struct {
	Username string
	Conn     *websocket.Conn
}

// Global variables
var (
	// Database (in-memory)
	messages   []Message
	messageID  int
	messageMux sync.RWMutex

	// WebSocket
	clients   = make(map[*Client]bool)
	broadcast = make(chan Message)
)

// InitDB initializes the in-memory storage
func InitDB() {
	log.Println("Using in-memory storage for messages")
	messages = make([]Message, 0)
	messageID = 1
}

// SaveMessage saves a message to memory
func SaveMessage(username, text string) error {
	messageMux.Lock()
	defer messageMux.Unlock()

	message := Message{
		ID:        messageID,
		Username:  username,
		Text:      text,
		Timestamp: time.Now(),
	}

	messages = append(messages, message)
	messageID++

	log.Printf("Message saved: %s: %s", username, text)
	return nil
}

// GetMessages retrieves messages from memory
func GetMessages(limit int) ([]Message, error) {
	messageMux.RLock()
	defer messageMux.RUnlock()

	if limit <= 0 {
		limit = 50 // Default limit
	}

	start := 0
	if len(messages) > limit {
		start = len(messages) - limit
	}

	// Return a copy to avoid race conditions
	result := make([]Message, len(messages)-start)
	copy(result, messages[start:])

	return result, nil
}

// GetMessageCount returns the total number of messages
func GetMessageCount() int {
	messageMux.RLock()
	defer messageMux.RUnlock()
	return len(messages)
}

// serveHome serves the HTML file
func serveHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

// handleConnections handles WebSocket connections
func handleConnections(ws *websocket.Conn) {
	defer ws.Close()

	// Read username from query parameter
	username := ws.Request().URL.Query().Get("username")
	if username == "" {
		username = "Anonymous"
	}

	client := &Client{Username: username, Conn: ws}
	clients[client] = true

	log.Printf("Client connected: %s (Total clients: %d)", username, len(clients))

	// Send message history to new client
	messages, err := GetMessages(100)
	if err == nil {
		for _, msg := range messages {
			websocket.JSON.Send(ws, msg)
		}
	}

	for {
		var msg Message
		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			log.Printf("Error reading JSON from %s: %v", username, err)
			delete(clients, client)
			log.Printf("Client disconnected: %s (Total clients: %d)", username, len(clients))
			break
		}

		msg.Username = username
		msg.Timestamp = time.Now()

		// Save to memory
		err = SaveMessage(msg.Username, msg.Text)
		if err != nil {
			log.Printf("Error saving message: %v", err)
		}

		// Broadcast to all clients
		broadcast <- msg
	}
}

// handleMessages processes incoming messages and broadcasts them
func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := websocket.JSON.Send(client.Conn, msg)
			if err != nil {
				log.Printf("WebSocket error for %s: %v", client.Username, err)
				client.Conn.Close()
				delete(clients, client)
				log.Printf("Client removed due to error: %s", client.Username)
			}
		}
	}
}

// getMessages returns the latest messages
func getMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := GetMessages(limit)
	if err != nil {
		http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// getStats returns basic statistics
func getStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_messages": GetMessageCount(),
		"active_clients": len(clients),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func main() {
	// Initialize in-memory storage
	InitDB()

	// Start message handler goroutine
	go handleMessages()

	// Setup HTTP routes
	http.HandleFunc("/", serveHome)
	http.Handle("/ws", websocket.Handler(handleConnections))
	http.HandleFunc("/messages", getMessages)
	http.HandleFunc("/stats", getStats)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	log.Println("Server starting on :8080")
	log.Println("Using in-memory storage - messages will be lost on server restart")
	log.Println("Visit http://localhost:8080 to access the chat")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
