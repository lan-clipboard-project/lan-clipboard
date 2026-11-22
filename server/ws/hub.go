package ws

import (
	"encoding/json"
	"log"
)

// Message represents a WebSocket message broadcast to all connected clients.
type Message struct {
	Type string      `json:"type"` // "clip_created", "clip_deleted", "clip_updated", "clips_cleared"
	Data interface{} `json:"data"` // payload (usually a Clip object or nil)
}

// Hub maintains the set of active WebSocket clients and broadcasts messages to them.
// It uses channels exclusively — no mutexes.
type Hub struct {
	clients    map[*Client]bool // registered clients
	register   chan *Client     // register requests
	unregister chan *Client     // unregister requests
	broadcast  chan []byte      // inbound messages to broadcast
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub's event loop. Should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("WebSocket: client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("WebSocket: client disconnected (total: %d)", len(h.clients))
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full; disconnect it
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// BroadcastMessage serializes a Message and sends it to all connected clients.
func (h *Hub) BroadcastMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: failed to marshal message: %v", err)
		return
	}
	h.broadcast <- data
}
