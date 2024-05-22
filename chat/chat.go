package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"sync"
)

// Client represents a single chatting user.
type Client struct {
	conn *websocket.Conn
	send chan []byte
	room *ChatRoom
}

// ChatRoom represents a chat room.
type ChatRoom struct {
	id         string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	closed     bool
	mu         sync.Mutex
}

// NewChatRoom creates a new chat room.
func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		id:         uuid.New().String(),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// GetID returns the unique ID of the chat room.
func (room *ChatRoom) GetID() string {
	return room.id
}

// Run starts the chat room to accept clients and messages.
func (room *ChatRoom) Run() {
	for {
		select {
		case client := <-room.register:
			room.clients[client] = true
			// Notify all clients about the new chat
			for c := range room.clients {
				select {
				case c.send <- []byte("New chat started"):
				default:
					close(c.send)
					delete(room.clients, c)
				}
			}
		case client := <-room.unregister:
			if _, ok := room.clients[client]; ok {
				delete(room.clients, client)
				close(client.send)
			}
		case message := <-room.broadcast:
			room.SaveMessage(message)
			for client := range room.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(room.clients, client)
				}
			}
		}
	}
}

// SaveMessage saves the chat message to a file.
func (room *ChatRoom) SaveMessage(message []byte) {
	f, err := os.OpenFile("chat_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(string(message) + "\n"); err != nil {
		log.Println(err)
	}
}

// Close closes the chat room.
func (room *ChatRoom) Close() {
	room.mu.Lock()
	defer room.mu.Unlock()
	room.closed = true
	// Notify all clients about the chat closing
	for client := range room.clients {
		select {
		case client.send <- []byte("Chat closed"):
		default:
			close(client.send)
			delete(room.clients, client)
		}
	}
}
