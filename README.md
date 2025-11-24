# Kaamos Server

Kaamos Server is a WebRTC SFU (Selective Forwarding Unit) designed for secure and efficient video calling. It handles media routing and signaling via WebSockets, following the specifications outlined in `Kaamos_ТЗ.md`.

## Features

- **WebRTC SFU**: Efficient media forwarding (VP8/H264 support).
- **WebSocket Signaling**: Real-time communication for session establishment.
- **Room Management**: Dynamic creation and management of video call rooms.
- **Secure**: Designed with security in mind (Key Exchange, Encrypted Data tunneling).

## Technology Stack

- **Language**: Golang 1.21+
- **WebRTC**: `github.com/pion/webrtc/v4`
- **WebSocket**: `github.com/gorilla/websocket`
- **Router**: `github.com/labstack/echo/v4`

## API Endpoints

### HTTP API

> **Note**: Some endpoints are currently in development and may return placeholder responses.

#### `GET /health`
Check the server status.
- **Response**: `200 OK`
  ```json
  {
    "status": "ok",
    "time": "2023-10-27T10:00:00Z"
  }
  ```

#### `POST /api/rooms/create`
Create a new room for a call.
- **Response**: `200 OK`
  ```json
  {
    "room_id": "abc123xyz",
    "url": "https://kaamos.yourdomain.com/room/abc123xyz",
    "expires_at": "2025-11-24T14:48:00Z"
  }
  ```

#### `GET /api/rooms/:room_id`
Get information about a specific room.
- **Response**: `200 OK`
  ```json
  {
    "room_id": "abc123xyz",
    "active": true,
    "participants": 1,
    "created_at": "2025-11-23T14:48:00Z"
  }
  ```

### WebSocket API

#### `GET /ws/:room_id`
Connect to the signaling server for a specific room.

**Signaling Protocol (JSON Messages):**

1.  **Join Room**
    ```json
    {
      "type": "join",
      "room_id": "abc123xyz",
      "user_id": "user_123"
    }
    ```

2.  **WebRTC Offer** (Client -> Server)
    ```json
    {
      "type": "offer",
      "data": {
        "sdp": "v=0..."
      }
    }
    ```

3.  **WebRTC Answer** (Server -> Client)
    ```json
    {
      "type": "answer",
      "data": {
        "sdp": "v=0..."
      }
    }
    ```

4.  **ICE Candidate**
    ```json
    {
      "type": "ice-candidate",
      "data": {
        "candidate": "...",
        "sdpMid": "0",
        "sdpMLineIndex": 0
      }
    }
    ```

5.  **Key Exchange** (For E2EE or secure signaling)
    ```json
    {
      "type": "key-exchange",
      "data": {
        "public_key": "..."
      }
    }
    ```

6.  **Encrypted Data** (Tunneling encrypted messages)
    ```json
    {
      "type": "encrypted",
      "data": {
        "to": "target_user_id",
        "payload": "..."
      }
    }
    ```

## Running the Server

1.  **Install Dependencies**:
    ```bash
    go mod download
    ```

2.  **Run**:
    ```bash
    go run main.go
    ```
    The server will start on port `8080` (default).

## Project Structure

```
kaamos-server/
├── main.go                    # Entry point
├── internal/
│   ├── app/                   # HTTP Handlers and App initialization
│   ├── signaling/             # WebRTC SFU and WebSocket logic
│   └── middleware/            # HTTP Middleware (Rate limiting, etc.)
├── Kaamos_ТЗ.md               # Technical Specification
└── README.md                  # This file
```
