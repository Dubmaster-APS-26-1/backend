package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
)

type Message struct {
	VideoID string `json:"videoId"`
	Status  string `json:"status"`
}

type Client struct {
	UserSsoID string
	Ch        chan Message
}

type Hub struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]bool)}
}

func (h *Hub) Register(userSsoID string) *Client {
	c := &Client{UserSsoID: userSsoID, Ch: make(chan Message, 32)}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	slog.Debug("[SSE] Client registered", "user", userSsoID, "total_clients", len(h.clients))
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.Ch)
	slog.Debug("[SSE] Client unregistered", "user", c.UserSsoID, "total_clients", len(h.clients))
}

func (h *Hub) Broadcast(videoID, status string) {
	msg := Message{VideoID: videoID, Status: status}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.Ch <- msg:
		default:
		}
	}
}

func (h *Hub) MarshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}
