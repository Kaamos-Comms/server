package signaling

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	rooms    map[string]*Room
	mutex    sync.RWMutex
	upgrader websocket.Upgrader
}

func NewServer() *Server {
	return &Server{
		rooms: make(map[string]*Room),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO : check origin in production
				return true
			},
		},
	}
}

func generateParticipantID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract roomID from path: /ws/{room_id}
	// Since we are using Echo's WrapHandler, we might not get the path param easily in r.URL.Path if it's rewritten?
	// Actually, r.URL.Path should be the full path.
	// Let's assume the last segment is the room ID.
	path := r.URL.Path
	// Simple split to get the last segment
	// Note: This is a bit brittle but works for /ws/:room_id
	var roomID string
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			roomID = path[i+1:]
			break
		}
	}

	if roomID == "" {
		http.Error(w, "Missing room ID", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Wait for Join message
	var joinMsg Message
	if err := conn.ReadJSON(&joinMsg); err != nil {
		log.Printf("Failed to read join message: %v", err)
		conn.Close()
		return
	}

	if joinMsg.Type != MessageTypeJoin {
		log.Printf("Expected join message, got: %s", joinMsg.Type)
		conn.Close()
		return
	}

	// Extract user_id from join message
	// The spec says: { "type": "join", "room_id": "...", "user_id": "..." }
	// We map this to our internal structures.
	// joinMsg.Data might be a map[string]interface{} because of JSON unmarshaling

	// Let's parse the data manually or assume it's in the message fields if we update Message struct?
	// The Message struct has `Data interface{}`.
	// Let's check `types.go`. Message has `Data interface{}`.
	// We need to cast it.

	data, ok := joinMsg.Data.(map[string]interface{})
	if !ok {
		// Try to re-marshal and unmarshal if needed, or just use the map
		// But wait, ReadJSON unmarshals into interface{}.
		// If we want strict typing we should define a JoinRequest struct.
	}

	userID, _ := data["user_id"].(string)
	if userID == "" {
		// Fallback or error? Spec says user_id is sent.
		userID = generateParticipantID()
	}

	participant := &Participant{
		ID:       userID,
		Conn:     NewWebSocketConnWrapper(conn),
		Status:   StatusConnected,
		JoinedAt: time.Now(),
	}

	s.joinRoom(roomID, participant)

	go s.handleConnection(roomID, participant)
}

func (s *Server) joinRoom(slug string, participant *Participant) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.rooms[slug]; !exists {
		s.rooms[slug] = NewRoom(slug)
	}

	room := s.rooms[slug]

	// Initialize SFU for the participant
	if err := s.initSFU(room, participant); err != nil {
		log.Printf("Failed to init SFU: %v", err)
		participant.Conn.Close()
		return
	}

	room.AddParticipant(participant)

	// Notify others?
	// For SFU, we might not need to broadcast "join" in the same way as P2P,
	// but we do need to handle track negotiation.
	// The spec says: Client -> Server: Join room.
	// Then Client -> Server: Offer.

	// We don't strictly need to send a response to Join if the client immediately sends Offer,
	// but sending a confirmation is good practice.
	// The spec doesn't explicitly show a Join response, but let's send one.

	participant.Conn.WriteJSON(&Message{
		Type:      MessageTypeJoin, // Ack
		RoomID:    slug,
		Timestamp: time.Now(),
	})
}

func (s *Server) handleConnection(slug string, participant *Participant) {
	defer s.leaveRoom(slug, participant)

	for {
		var message Message
		err := participant.Conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Read message error: %v", err)
			break
		}

		message.From = participant.ID
		message.RoomID = slug
		message.Timestamp = time.Now()

		s.handleMessage(slug, participant, &message)
	}
}

func (s *Server) leaveRoom(slug string, participant *Participant) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	room, exists := s.rooms[slug]
	if !exists {
		return
	}

	room.RemovePublicKey(participant.ID)

	room.RemoveParticipant(participant.ID)
	participant.Conn.Close()

	leaveMessage := &Message{
		Type:      MessageTypeLeave,
		From:      participant.ID,
		RoomID:    slug,
		Timestamp: time.Now(),
	}
	room.BroadcastToAll(leaveMessage, participant.ID)

	room.BroadcastPublicKeys(participant.ID)

	if room.IsEmpty() {
		delete(s.rooms, slug)
		log.Printf("Room %s deleted (empty)", slug)
	}
}

func (s *Server) GetRoomStats(slug string) map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	room, exists := s.rooms[slug]
	if !exists {
		return nil
	}

	return map[string]interface{}{
		"slug":         room.Slug,
		"participants": room.GetParticipantsData(),
		"created_at":   room.CreatedAt,
		"has_host":     room.Host != nil,
		"guests_count": len(room.Guests),
	}
}

func (s *Server) Shutdown() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, room := range s.rooms {
		if room.Host != nil {
			room.Host.Conn.Close()
		}
		for _, guest := range room.Guests {
			guest.Conn.Close()
		}
	}
	s.rooms = make(map[string]*Room)
}
