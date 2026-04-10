package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Event is a WebSocket message sent to connected clients.
type Event struct {
	Type    string      `json:"type"`
	Project string      `json:"project"`
	Cmd     string      `json:"cmd,omitempty"`
	Data    interface{} `json:"data"`
}

// clientMessage is a message received from a WebSocket client.
type clientMessage struct {
	Type    string `json:"type"`
	Project string `json:"project"`
}

// client wraps a WebSocket connection with its subscriptions.
type client struct {
	conn         *websocket.Conn
	mu           sync.Mutex // protects writes to conn
	projects     map[string]bool
	projectsMu   sync.RWMutex
}

// writeJSON sends a JSON message to the client in a thread-safe way.
func (c *client) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

// subscribedTo checks if the client is subscribed to a given project.
// An empty subscriptions map means the client receives all events.
func (c *client) subscribedTo(project string) bool {
	c.projectsMu.RLock()
	defer c.projectsMu.RUnlock()
	if len(c.projects) == 0 {
		return true
	}
	return c.projects[project]
}

// subscribe adds a project to the client's subscriptions.
func (c *client) subscribe(project string) {
	c.projectsMu.Lock()
	defer c.projectsMu.Unlock()
	c.projects[project] = true
}

// unsubscribe removes a project from the client's subscriptions.
func (c *client) unsubscribe(project string) {
	c.projectsMu.Lock()
	defer c.projectsMu.Unlock()
	delete(c.projects, project)
}

// Hub manages WebSocket connections and broadcasts events.
type Hub struct {
	clients    map[*client]bool
	mu         sync.RWMutex
	broadcast  chan Event
	register   chan *client
	unregister chan *client
	done       chan struct{}
}

// NewHub creates and starts a new WebSocket hub.
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *client),
		unregister: make(chan *client),
		done:       make(chan struct{}),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case <-h.done:
			h.mu.Lock()
			for c := range h.clients {
				c.conn.Close()
				delete(h.clients, c)
			}
			h.mu.Unlock()
			return

		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				c.conn.Close()
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			var dead []*client
			h.mu.RLock()
			for c := range h.clients {
				if !c.subscribedTo(event.Project) {
					continue
				}
				if err := c.writeJSON(event); err != nil {
					log.Printf("ws write error: %v", err)
					dead = append(dead, c)
				}
			}
			h.mu.RUnlock()
			if len(dead) > 0 {
				h.mu.Lock()
				for _, c := range dead {
					delete(h.clients, c)
					c.conn.Close()
				}
				h.mu.Unlock()
			}
		}
	}
}

// Close stops the hub's run loop and closes all client connections.
func (h *Hub) Close() {
	close(h.done)
}

// Broadcast sends an event to all subscribed clients.
func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev mode
	},
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	c := &client{
		conn:     conn,
		projects: make(map[string]bool),
	}

	h.register <- c

	// Read messages from the client for subscribe/unsubscribe.
	go func() {
		defer func() { h.unregister <- c }()
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg clientMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "subscribe":
				if msg.Project != "" {
					c.subscribe(msg.Project)
					log.Printf("ws client subscribed to project: %s", msg.Project)
				}
			case "unsubscribe":
				if msg.Project != "" {
					c.unsubscribe(msg.Project)
					log.Printf("ws client unsubscribed from project: %s", msg.Project)
				}
			}
		}
	}()
}
