package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
}

type NotificationEvent struct {
	UserID  string      `json:"user_id"`
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Payload interface{} `json:"payload,omitempty"`
}

type Hub struct {
	clients    map[string][]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan NotificationEvent
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string][]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan NotificationEvent),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = append(h.clients[client.UserID], client)
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			clients := h.clients[client.UserID]
			for i, c := range clients {
				if c == client {
					clients = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			h.clients[client.UserID] = clients
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.Lock()
			clients := h.clients[event.UserID]
			for _, c := range clients {
				err := c.Conn.WriteJSON(event)
				if err != nil {
					log.Println("Write error:", err)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Emit(event NotificationEvent) {
	h.broadcast <- event
}
