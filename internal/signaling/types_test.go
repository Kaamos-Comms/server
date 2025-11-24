package signaling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessageType(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"Join message", MessageTypeJoin, "join"},
		{"Leave message", MessageTypeLeave, "leave"},
		{"Knock message", MessageTypeKnock, "knock"},
		{"Allow message", MessageTypeAllow, "allow"},
		{"Deny message", MessageTypeDeny, "deny"},
		{"Offer message", MessageTypeOffer, "offer"},
		{"Answer message", MessageTypeAnswer, "answer"},
		{"ICE candidate", MessageTypeICECandidate, "ice_candidate"},
		{"Participants", MessageTypeParticipants, "participants"},
		{"Error message", MessageTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.msgType))
		})
	}
}

func TestParticipantRole(t *testing.T) {
	assert.Equal(t, "host", string(RoleHost))
	assert.Equal(t, "guest", string(RoleGuest))
}

func TestParticipantStatus(t *testing.T) {
	assert.Equal(t, "connected", string(StatusConnected))
	assert.Equal(t, "knocking", string(StatusKnocking))
	assert.Equal(t, "in_room", string(StatusInRoom))
	assert.Equal(t, "disconnected", string(StatusDisconnected))
}

func TestParticipantCreation(t *testing.T) {
	participant := &Participant{
		ID:       "test-id",
		Role:     RoleHost,
		Status:   StatusConnected,
		Name:     "Test User",
		JoinedAt: time.Now(),
	}

	assert.Equal(t, "test-id", participant.ID)
	assert.Equal(t, RoleHost, participant.Role)
	assert.Equal(t, StatusConnected, participant.Status)
	assert.Equal(t, "Test User", participant.Name)
	assert.WithinDuration(t, time.Now(), participant.JoinedAt, time.Second)
}

func TestMessage(t *testing.T) {
	message := &Message{
		Type:      MessageTypeJoin,
		From:      "user1",
		To:        "user2",
		RoomID:    "test-room-123",
		Data:      "test data",
		Timestamp: time.Now(),
	}

	assert.Equal(t, MessageTypeJoin, message.Type)
	assert.Equal(t, "user1", message.From)
	assert.Equal(t, "user2", message.To)
	assert.Equal(t, "test-room-123", message.RoomID)
	assert.Equal(t, "test data", message.Data)
	assert.WithinDuration(t, time.Now(), message.Timestamp, time.Second)
}

func TestJoinData(t *testing.T) {
	joinData := &JoinData{
		Name: "Test User",
		Role: RoleGuest,
	}

	assert.Equal(t, "Test User", joinData.Name)
	assert.Equal(t, RoleGuest, joinData.Role)
}

func TestErrorData(t *testing.T) {
	errorData := &ErrorData{
		Code:    "INVALID_ROLE",
		Message: "Role must be host or guest",
	}

	assert.Equal(t, "INVALID_ROLE", errorData.Code)
	assert.Equal(t, "Role must be host or guest", errorData.Message)
}
