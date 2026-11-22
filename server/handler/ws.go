package handler

import (
	"log"
	"net/http"
	"net/url"

	"lan-clipboard/middleware"
	wslib "lan-clipboard/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub      *wslib.Hub
	upgrader websocket.Upgrader
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *wslib.Hub, publicOrigin *url.URL) *WSHandler {
	return &WSHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return middleware.BrowserOriginAllowed(r, publicOrigin)
			},
		},
	}
}

// Handle upgrades the HTTP connection to WebSocket and registers the client.
func (h *WSHandler) Handle(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket: upgrade failed: %v", err)
		return
	}

	client := wslib.NewClient(h.hub, conn)
	go client.WritePump()
	go client.ReadPump()
}
