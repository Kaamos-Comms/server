package signaling

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRoom(t *testing.T) {
	slug := "test-room"
	room := NewRoom(slug)

	require.Equal(t, slug, room.Slug)
	require.NotNil(t, room.Guests)
	require.NotNil(t, room.PublicKeys)
	require.Nil(t, room.Host)
}

func TestAddParticipant(t *testing.T) {
	room := NewRoom("test-room")
	host := &Participant{ID: "host", Role: RoleHost}
	guest := &Participant{ID: "guest", Role: RoleGuest}

	err := room.AddParticipant(host)
	require.NoError(t, err)
	require.Equal(t, host, room.Host)
	require.Equal(t, StatusInRoom, host.Status)

	err = room.AddParticipant(guest)
	require.NoError(t, err)
	require.Equal(t, guest, room.Guests["guest"])
	require.Equal(t, StatusKnocking, guest.Status)
}

func TestRemoveParticipant(t *testing.T) {
	room := NewRoom("test-room")
	host := &Participant{ID: "host", Role: RoleHost}
	guest := &Participant{ID: "guest", Role: RoleGuest}

	room.AddParticipant(host)
	room.AddParticipant(guest)

	room.RemoveParticipant("guest")
	require.Nil(t, room.Guests["guest"])

	room.RemoveParticipant("host")
	require.Nil(t, room.Host)
}
