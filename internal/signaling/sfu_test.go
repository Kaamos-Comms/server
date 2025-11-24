package signaling

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitSFU(t *testing.T) {
	server := NewServer()
	room := NewRoom("test-room")
	participant := &Participant{ID: "user1"}

	// We can't easily test the full WebRTC connection without a real network or mock,
	// but we can test that initSFU creates a PeerConnection and assigns it to the participant.

	// Note: This might fail if the environment doesn't support WebRTC networking (e.g. no network interface),
	// but usually it works in CI environments.
	err := server.initSFU(room, participant)
	require.NoError(t, err)
	require.NotNil(t, participant.PC)

	// Cleanup
	if participant.PC != nil {
		participant.PC.Close()
	}
}
