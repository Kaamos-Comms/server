package signaling

import (
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleWebRTCMessageSFU(t *testing.T) {
	server := NewServer()
	room := NewRoom("test-room")

	mockGuestConn := &MockWebSocketConn{}
	mockHostConn := &MockWebSocketConn{}

	// Create real PeerConnection for guest
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	assert.NoError(t, err)
	defer pc.Close()

	guest := &Participant{
		ID:     "guest1",
		Conn:   mockGuestConn,
		Role:   RoleGuest,
		Status: StatusInRoom,
		PC:     pc,
	}

	host := &Participant{
		ID:     "host1",
		Conn:   mockHostConn,
		Role:   RoleHost,
		Status: StatusInRoom,
	}

	room.AddParticipant(host)
	room.AddParticipant(guest)

	// Create a valid SDP offer
	// offer, err := pc.CreateOffer(nil)
	// assert.NoError(t, err)

	// We need to set local description on the PC so it's in the right state?
	// Actually, the handler calls SetRemoteDescription.
	// So we need an offer from *another* PC or just a dummy one.
	// Let's use the one we just created, but we need to be careful about state.
	// If we send this offer to the handler, the handler will call SetRemoteDescription on `guest.PC`.
	// But `guest.PC` created the offer, so it expects SetLocalDescription.
	// We should create a *remote* PC to generate the offer.

	remotePC, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	assert.NoError(t, err)
	defer remotePC.Close()

	_, err = remotePC.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo)
	assert.NoError(t, err)

	remoteOffer, err := remotePC.CreateOffer(nil)
	assert.NoError(t, err)

	webRTCMessage := &Message{
		Type: MessageTypeOffer,
		Data: map[string]interface{}{"sdp": remoteOffer.SDP},
	}

	// Expect Answer to be sent back to guest
	mockGuestConn.On("WriteJSON", mock.MatchedBy(func(msg *Message) bool {
		return msg.Type == MessageTypeAnswer
	})).Return(nil).Once()

	// Host should NOT receive anything
	mockHostConn.AssertNotCalled(t, "WriteJSON")

	server.handleWebRTCMessage(room, guest, webRTCMessage)

	// Wait a bit for async processing if any (handleWebRTCMessage is synchronous in logic but might have async parts? No, it's sync)
	// But WriteJSON is called synchronously.

	mockGuestConn.AssertExpectations(t)
	mockHostConn.AssertExpectations(t)
}

func TestHandleWebRTCMessageNoPC(t *testing.T) {
	server := NewServer()
	room := NewRoom("test-room")

	mockGuestConn := &MockWebSocketConn{}
	guest := &Participant{
		ID:     "guest1",
		Conn:   mockGuestConn,
		Role:   RoleGuest,
		Status: StatusInRoom,
		PC:     nil, // No PC
	}

	room.AddParticipant(guest)

	message := &Message{
		Type: MessageTypeOffer,
		Data: map[string]interface{}{"sdp": "test-offer"},
	}

	server.handleWebRTCMessage(room, guest, message)

	// No messages should have been sent
	mockGuestConn.AssertNotCalled(t, "WriteJSON")
}

func TestHandleAllow(t *testing.T) {
	server := NewServer()
	room := NewRoom("test-room")
	server.rooms["test-room"] = room

	mockHostConn := &MockWebSocketConn{}
	mockGuestConn := &MockWebSocketConn{}

	host := &Participant{
		ID:     "host1",
		Conn:   mockHostConn,
		Role:   RoleHost,
		Status: StatusInRoom,
	}

	guest := &Participant{
		ID:     "guest1",
		Conn:   mockGuestConn,
		Role:   RoleGuest,
		Status: StatusKnocking,
	}

	room.AddParticipant(host)
	room.AddParticipant(guest)

	message := &Message{
		Type: MessageTypeAllow,
		Data: "guest1",
	}

	mockGuestConn.On("WriteJSON", mock.Anything).Return(nil).Times(2) // Allow + Participants
	mockHostConn.On("WriteJSON", mock.Anything).Return(nil).Once()    // Participants

	server.handleAllow(room, host, message)

	assert.Equal(t, StatusInRoom, guest.Status)
	mockGuestConn.AssertExpectations(t)
	mockHostConn.AssertExpectations(t)
}

func TestHandleDeny(t *testing.T) {
	server := NewServer()
	room := NewRoom("test-room")

	mockHostConn := &MockWebSocketConn{}
	mockGuestConn := &MockWebSocketConn{}

	host := &Participant{
		ID:   "host1",
		Conn: mockHostConn,
		Role: RoleHost,
	}

	guest := &Participant{
		ID:     "guest1",
		Conn:   mockGuestConn,
		Role:   RoleGuest,
		Status: StatusKnocking,
	}

	room.AddParticipant(host)
	room.AddParticipant(guest)

	message := &Message{
		Type: MessageTypeDeny,
		Data: "guest1",
	}

	// Expect deny message and close
	mockGuestConn.On("WriteJSON", mock.MatchedBy(func(msg *Message) bool {
		return msg.Type == MessageTypeDeny
	})).Return(nil).Once()
	mockGuestConn.On("Close").Return(nil).Once()

	server.handleDeny(room, host, message)

	assert.Equal(t, 0, len(room.Guests))
	mockGuestConn.AssertExpectations(t)
}
