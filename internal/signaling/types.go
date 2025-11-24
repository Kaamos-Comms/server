package signaling

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

const (
	RoleHost  ParticipantRole = "host"
	RoleGuest ParticipantRole = "guest"

	MessageTypeJoin         MessageType = "join"
	MessageTypeLeave        MessageType = "leave"
	MessageTypeKnock        MessageType = "knock"
	MessageTypeAllow        MessageType = "allow"
	MessageTypeDeny         MessageType = "deny"
	MessageTypeOffer        MessageType = "offer"
	MessageTypeAnswer       MessageType = "answer"
	MessageTypeICECandidate MessageType = "ice_candidate"
	MessageTypeParticipants MessageType = "participants"
	MessageTypeError        MessageType = "error"
	MessageTypeKeyExchange  MessageType = "key_exchange"
	MessageTypePublicKeys   MessageType = "public_keys"
	MessageTypeEncrypted    MessageType = "encrypted_data"

	StatusConnected    ParticipantStatus = "connected"
	StatusKnocking     ParticipantStatus = "knocking"
	StatusInRoom       ParticipantStatus = "in_room"
	StatusDisconnected ParticipantStatus = "disconnected"
)

type ParticipantKeys struct {
	PublicKey string `json:"public_key"` // Base64-encoded public key
}

type Participant struct {
	ID       string                        `json:"id"`
	Conn     WebSocketConnInterface        `json:"-"`
	Role     ParticipantRole               `json:"role"`
	Status   ParticipantStatus             `json:"status"`
	Name     string                        `json:"name,omitempty"`
	Keys     ParticipantKeys               `json:"keys,omitempty"`
	JoinedAt time.Time                     `json:"joined_at"`
	PC       *webrtc.PeerConnection        `json:"-"`
	Tracks   []*webrtc.TrackLocalStaticRTP `json:"-"`
}

type Room struct {
	Slug       string                        `json:"slug"`
	Host       *Participant                  `json:"host,omitempty"`
	Guests     map[string]*Participant       `json:"guests"`
	PublicKeys map[string]string             `json:"public_keys"`
	Tracks     []*webrtc.TrackLocalStaticRTP `json:"-"`
	CreatedAt  time.Time                     `json:"created_at"`
	mutex      sync.RWMutex
}

type KeyExchangeData struct {
	PublicKey string `json:"public_key"`
}

type PublicKeysData struct {
	Keys map[string]string `json:"keys"` // participantID -> publicKey
}

type EncryptedData struct {
	To        string `json:"to"`        // ID получателя
	Data      string `json:"data"`      // Base64-кодированные зашифрованные данные
	Algorithm string `json:"algorithm"` // "ed25519" или другой алгоритм
}

type MessageType string

type ParticipantRole string

type ParticipantStatus string

type WebSocketConnInterface interface {
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
	Close() error
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
}

type WebSocketConnWrapper struct {
	*websocket.Conn
}

func NewWebSocketConnWrapper(conn *websocket.Conn) *WebSocketConnWrapper {
	return &WebSocketConnWrapper{Conn: conn}
}

type Message struct {
	Type      MessageType `json:"type"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	RoomID    string      `json:"room_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type JoinData struct {
	Name string          `json:"name"`
	Role ParticipantRole `json:"role"`
}

type ParticipantsData struct {
	Host   *Participant            `json:"host,omitempty"`
	Guests map[string]*Participant `json:"guests"`
	Count  int                     `json:"count"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
