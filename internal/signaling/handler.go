package signaling

import (
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

func (s *Server) handleMessage(slug string, participant *Participant, message *Message) {
	s.mutex.RLock()
	room, exists := s.rooms[slug]
	s.mutex.RUnlock()

	if !exists {
		return
	}

	switch message.Type {
	case MessageTypeAllow:
		s.handleAllow(room, participant, message)
	case MessageTypeDeny:
		s.handleDeny(room, participant, message)
	case MessageTypeOffer, MessageTypeAnswer, MessageTypeICECandidate:
		s.handleWebRTCMessage(room, participant, message)
	case MessageTypeKeyExchange:
		s.handleKeyExchange(room, participant, message)
	case MessageTypeEncrypted:
		s.handleEncryptedData(room, participant, message)
	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

func (s *Server) handleKeyExchange(room *Room, participant *Participant, message *Message) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid key exchange data format")
		return
	}

	publicKey, ok := data["public_key"].(string)
	if !ok {
		log.Printf("Missing public key in key exchange")
		return
	}

	if err := room.SavePublicKey(participant.ID, publicKey); err != nil {
		log.Printf("Failed to save public key for %s: %v", participant.ID, err)

		errorMsg := &Message{
			Type: MessageTypeError,
			Data: ErrorData{
				Code:    "INVALID_PUBLIC_KEY",
				Message: "Invalid public key format",
			},
			Timestamp: time.Now(),
		}
		participant.Conn.WriteJSON(errorMsg)
		return
	}

	log.Printf("Saved public key for participant %s in room %s", participant.ID, room.Slug)

	// ВАЖНО: Убедитесь что BroadcastPublicKeys вызывается
	log.Printf("Broadcasting public keys to all participants")
	room.BroadcastPublicKeys("")
	log.Printf("Broadcast completed")
}

func (s *Server) handleEncryptedData(room *Room, participant *Participant, message *Message) {
	if participant.Status != StatusInRoom {
		return
	}

	data, ok := message.Data.(map[string]interface{})
	if !ok {
		return
	}

	toParticipantID, ok := data["to"].(string)
	if !ok {
		return
	}

	message.From = participant.ID
	message.Timestamp = time.Now()

	if toParticipantID == "all" {
		room.BroadcastToAll(message, participant.ID)
	} else {
		if room.Host != nil && room.Host.ID == toParticipantID {
			room.BroadcastToHost(message)
		} else {
			room.BroadcastToGuest(toParticipantID, message)
		}
	}
}

func (s *Server) handleAllow(room *Room, participant *Participant, message *Message) {
	// Only the host can allow
	if participant.Role != RoleHost {
		return
	}

	guestID, ok := message.Data.(string)
	if !ok {
		return
	}

	err := room.AllowGuest(guestID)
	if err != nil {
		log.Printf("Failed to allow guest: %v", err)
		return
	}

	// Notify the guest about the allowance
	allowMessage := &Message{
		Type:      MessageTypeAllow,
		From:      participant.ID,
		To:        guestID,
		RoomID:    room.Slug,
		Timestamp: time.Now(),
	}
	room.BroadcastToGuest(guestID, allowMessage)

	// Notify all participants about the updated participant list
	participantsMessage := &Message{
		Type:      MessageTypeParticipants,
		RoomID:    room.Slug,
		Data:      room.GetParticipantsData(),
		Timestamp: time.Now(),
	}
	room.BroadcastToAll(participantsMessage, "")
}

func (s *Server) handleDeny(room *Room, participant *Participant, message *Message) {
	// Only the host can deny
	if participant.Role != RoleHost {
		return
	}

	guestID, ok := message.Data.(string)
	if !ok {
		return
	}

	guest := room.GetParticipant(guestID)
	if guest == nil {
		return
	}

	// Notify the guest about the denial
	denyMessage := &Message{
		Type:      MessageTypeDeny,
		From:      participant.ID,
		To:        guestID,
		RoomID:    room.Slug,
		Timestamp: time.Now(),
	}
	room.BroadcastToGuest(guestID, denyMessage)

	// Remove the guest from the room
	room.DenyGuest(guestID)
	guest.Conn.Close()
}

func (s *Server) handleWebRTCMessage(room *Room, participant *Participant, message *Message) {
	if participant.PC == nil {
		return
	}

	data, ok := message.Data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid data for WebRTC message")
		return
	}

	switch message.Type {
	case MessageTypeOffer:
		sdpStr, _ := data["sdp"].(string)
		if err := participant.PC.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  sdpStr,
		}); err != nil {
			log.Printf("Failed to set remote description: %v", err)
			return
		}

		answer, err := participant.PC.CreateAnswer(nil)
		if err != nil {
			log.Printf("Failed to create answer: %v", err)
			return
		}

		if err := participant.PC.SetLocalDescription(answer); err != nil {
			log.Printf("Failed to set local description: %v", err)
			return
		}

		participant.Conn.WriteJSON(&Message{
			Type:      MessageTypeAnswer,
			RoomID:    room.Slug,
			Data:      map[string]interface{}{"sdp": answer.SDP, "type": answer.Type.String()},
			Timestamp: time.Now(),
		})

	case MessageTypeAnswer:
		sdpStr, _ := data["sdp"].(string)
		if err := participant.PC.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  sdpStr,
		}); err != nil {
			log.Printf("Failed to set remote description (answer): %v", err)
		}

	case MessageTypeICECandidate:
		candidateStr, _ := data["candidate"].(string)
		sdpMid, _ := data["sdpMid"].(string)
		sdpMLineIndex, _ := data["sdpMLineIndex"].(float64)

		// Convert float64 to uint16 safely
		lineIndex := uint16(sdpMLineIndex)

		candidate := webrtc.ICECandidateInit{
			Candidate:     candidateStr,
			SDPMid:        &sdpMid,
			SDPMLineIndex: &lineIndex,
		}

		if err := participant.PC.AddICECandidate(candidate); err != nil {
			log.Printf("Failed to add ICE candidate: %v", err)
		}
	}
}
