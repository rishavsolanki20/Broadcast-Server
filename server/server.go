//server.go

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// CricketScore represents the score details for a cricket match
type CricketScore struct {
	MatchID    string `json:"match_id"`
	TeamA      string `json:"team_a"`
	TeamB      string `json:"team_b"`
	ScoreA     string `json:"score_a"`
	ScoreB     string `json:"score_b"`
	OversA     string `json:"overs_a"`
	OversB     string `json:"overs_b"`
	Commentary string `json:"commentary"`
}

// Server struct
type Server struct {
	clients   map[*websocket.Conn]bool   // Connected clients
	broadcast chan []byte                // Broadcast channel for messages
	mu        sync.Mutex                 // Mutex for thread-safe access
	peers     map[string]*websocket.Conn // Map of peer addresses to connections
}

// NewServer initializes a new server
func NewServer() *Server {
	return &Server{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
		peers:     make(map[string]*websocket.Conn),
	}
}

// Handle client connections
func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v\n", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	log.Println("Client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Client disconnected")
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
			break
		}
		s.broadcast <- msg
	}
}

// Handle peer-to-peer connections
func (s *Server) handlePeerConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading peer connection: %v\n", err)
		return
	}
	defer conn.Close()

	log.Println("Peer connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Peer disconnected")
			return
		}
		s.broadcast <- msg
	}
}

// Connect to another peer server
func (s *Server) connectToPeer(peerAddress string) {
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+peerAddress+"/peer", nil)
	if err != nil {
		log.Printf("Error connecting to peer %s: %v\n", peerAddress, err)
		return
	}
	s.mu.Lock()
	s.peers[peerAddress] = conn
	s.mu.Unlock()

	log.Printf("Connected to peer: %s\n", peerAddress)

	go func() {
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Disconnected from peer: %s\n", peerAddress)
				s.mu.Lock()
				delete(s.peers, peerAddress)
				s.mu.Unlock()
				return
			}
			s.broadcast <- msg
		}
	}()
}

// Handle broadcasts to all clients and peers
func (s *Server) handleBroadcasts() {
	for {
		msg := <-s.broadcast
		s.mu.Lock()
		// Broadcast to clients
		for client := range s.clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("Error broadcasting to client: %v\n", err)
				client.Close()
				delete(s.clients, client)
			}
		}
		// Broadcast to peers
		for _, peerConn := range s.peers {
			err := peerConn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("Error broadcasting to peer: %v\n", err)
				peerConn.Close()
			}
		}
		s.mu.Unlock()
	}
}

// UpdateAndBroadcastScore updates the cricket score and broadcasts it
func (s *Server) UpdateAndBroadcastScore(score CricketScore) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert the CricketScore to JSON
	msg, err := json.Marshal(score)
	if err != nil {
		log.Printf("Error marshaling score: %v\n", err)
		return
	}

	// Send the JSON message to the broadcast channel
	s.broadcast <- msg
}

// Handle admin updates
func (s *Server) handleAdminUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var score CricketScore
	err := json.NewDecoder(r.Body).Decode(&score)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Update and broadcast the score
	s.UpdateAndBroadcastScore(score)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Score updated and broadcasted"))
}

// Broadcast cricket scores periodically
func (s *Server) broadcastScores() {
	for {
		// Simulated cricket score update
		score := []byte("Live Score: India 120/3 in 15 overs")
		s.mu.Lock()
		for client := range s.clients {
			log.Printf("Broadcasting to client: %v\n", client)
			err := client.WriteMessage(websocket.TextMessage, score)
			if err != nil {
				log.Printf("Error broadcasting to client: %v\n", err)
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mu.Unlock()

		// Wait for 10 seconds before the next update
		time.Sleep(10 * time.Second)
	}
}

// Start the server
func (s *Server) Start(port string, peers []string) {
	http.HandleFunc("/ws", s.handleConnections)
	http.HandleFunc("/peer", s.handlePeerConnections)
	http.HandleFunc("/admin", s.handleAdminUpdates)
	// Connect to peers
	for _, peer := range peers {
		go s.connectToPeer(peer)
	}

	// Start broadcast handler
	go s.handleBroadcasts()

	log.Printf("Server started on :%s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}
