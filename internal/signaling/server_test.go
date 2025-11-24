package signaling

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestShutdown(t *testing.T) {
	server := NewServer()
	slug := "test-room"

	mockHostConn := &MockWebSocketConn{}
	mockGuestConn := &MockWebSocketConn{}

	host := &Participant{
		ID:   "host1",
		Conn: mockHostConn,
		Role: RoleHost,
	}

	guest := &Participant{
		ID:   "guest1",
		Conn: mockGuestConn,
		Role: RoleGuest,
	}

	// Create room and add participants
	server.rooms[slug] = NewRoom(slug)
	server.rooms[slug].AddParticipant(host)
	server.rooms[slug].AddParticipant(guest)

	assert.Equal(t, 1, len(server.rooms))

	// Wait for Close calls on both connections
	mockHostConn.On("Close").Return(nil).Once()
	mockGuestConn.On("Close").Return(nil).Once()

	server.Shutdown()

	// Check that all rooms are cleared after shutdown
	assert.Equal(t, 0, len(server.rooms))

	mockHostConn.AssertExpectations(t)
	mockGuestConn.AssertExpectations(t)
}

func TestWebSocketConnection(t *testing.T) {
	server := NewServer()

	testServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer testServer.Close()

	// Construct URL with room ID in path: /ws/test-room
	// httptest server URL is http://127.0.0.1:port
	// We need ws://127.0.0.1:port/ws/test-room
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws/test-room"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Send Join message
	joinMsg := Message{
		Type: MessageTypeJoin,
		Data: map[string]interface{}{
			"user_id": "user1",
		},
	}
	err = conn.WriteJSON(joinMsg)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	stats := server.GetRoomStats("test-room")
	assert.NotNil(t, stats)
	assert.Equal(t, "test-room", stats["slug"])
	// Since we didn't specify role in Join (it defaults to something or logic handles it),
	// let's check if participant is added.
	// Actually, the current implementation in server.go doesn't set role from Join message yet,
	// it might default to Guest or Host depending on logic I wrote?
	// Let's check server.go logic.
	// It creates participant with ID from message.
	// It calls joinRoom.
	// joinRoom calls AddParticipant.
	// AddParticipant checks Role. If RoleHost, sets Host. Else adds to Guests.
	// Participant struct default Role is empty string.
	// So it will be added as Guest (StatusKnocking).

	assert.Equal(t, 1, stats["guests_count"])
}

func TestNewServer(t *testing.T) {
	server := NewServer()
	assert.NotNil(t, server)
	assert.NotNil(t, server.rooms)
	assert.Equal(t, 0, len(server.rooms))
}

func TestInvalidWebSocketParams(t *testing.T) {
	server := NewServer()

	// Missing room ID in path
	req := httptest.NewRequest(http.MethodGet, "/ws/", nil)
	w := httptest.NewRecorder()

	server.HandleWebSocket(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
