package signaling

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

func (s *Server) initSFU(room *Room, participant *Participant) error {
	// Create PeerConnection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}

	participant.PC = pc

	// Handle ICE candidates
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidateJSON := c.ToJSON()
		participant.Conn.WriteJSON(&Message{
			Type:      MessageTypeICECandidate,
			RoomID:    room.Slug,
			Data:      candidateJSON,
			Timestamp: time.Now(),
		})
	})

	// Handle incoming tracks
	pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Track received from %s: %s %s", participant.ID, remoteTrack.ID(), remoteTrack.Kind())

		// Create a local track to forward to other participants
		localTrack, err := webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, remoteTrack.ID(), remoteTrack.StreamID())
		if err != nil {
			log.Printf("Failed to create local track: %v", err)
			return
		}

		participant.Tracks = append(participant.Tracks, localTrack)
		room.Tracks = append(room.Tracks, localTrack)

		// Forward media packets
		go func() {
			rtpBuf := make([]byte, 1400)
			for {
				i, _, err := remoteTrack.Read(rtpBuf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Failed to read from remote track: %v", err)
					}
					return
				}

				if _, err = localTrack.Write(rtpBuf[:i]); err != nil {
					if err != io.ErrClosedPipe {
						log.Printf("Failed to write to local track: %v", err)
					}
					return
				}
			}
		}()

		// Add this new track to all OTHER participants
		s.addTrackToParticipants(room, participant.ID, localTrack)
	})

	// Add existing tracks from other participants to this new participant
	for _, otherParticipant := range room.Guests {
		if otherParticipant.ID == participant.ID {
			continue
		}
		for _, track := range otherParticipant.Tracks {
			if _, err := pc.AddTrack(track); err != nil {
				log.Printf("Failed to add track to new participant: %v", err)
			}
		}
	}
	if room.Host != nil && room.Host.ID != participant.ID {
		for _, track := range room.Host.Tracks {
			if _, err := pc.AddTrack(track); err != nil {
				log.Printf("Failed to add host track to new participant: %v", err)
			}
		}
	}

	return nil
}

func (s *Server) addTrackToParticipants(room *Room, sourceID string, track *webrtc.TrackLocalStaticRTP) {
	// Helper to add track to a participant
	add := func(p *Participant) {
		if p.ID == sourceID || p.PC == nil {
			return
		}

		if _, err := p.PC.AddTrack(track); err != nil {
			log.Printf("Failed to add track to participant %s: %v", p.ID, err)
			return
		}

		// Renegotiate?
		// In a simple SFU, we might need to trigger a new Offer from the server side,
		// or just let the client handle "negotiationneeded".
		// For MVP, let's assume the client handles onnegotiationneeded or we send an offer.
		// But Pion doesn't automatically trigger negotiation.
		// We should create an Offer and send it to the participant.

		offer, err := p.PC.CreateOffer(nil)
		if err != nil {
			log.Printf("Failed to create offer for renegotiation: %v", err)
			return
		}

		if err := p.PC.SetLocalDescription(offer); err != nil {
			log.Printf("Failed to set local description: %v", err)
			return
		}

		p.Conn.WriteJSON(&Message{
			Type:      MessageTypeOffer,
			RoomID:    room.Slug,
			Data:      map[string]interface{}{"sdp": offer.SDP, "type": offer.Type.String()},
			Timestamp: time.Now(),
		})
	}

	if room.Host != nil {
		add(room.Host)
	}
	for _, guest := range room.Guests {
		add(guest)
	}
}
