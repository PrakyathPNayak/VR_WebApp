package websocket

import (
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
    "VR-Distributed/internal/crypto"
    "VR-Distributed/internal/webrtc"
    "VR-Distributed/pkg/types"
)

var (
    upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
    }
    
    rooms      = make(map[string]*Room)
    roomsMutex = sync.RWMutex{}
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket upgrade error:", err)
        return
    }
    defer conn.Close()

    peerID := r.URL.Query().Get("peer_id")
    roomID := r.URL.Query().Get("room")
    
    if peerID == "" {
        peerID = fmt.Sprintf("peer_%d", time.Now().UnixNano())
    }
    if roomID == "" {
        roomID = "default"
    }

    client := NewClient(conn, peerID, roomID)
    
    // Setup WebRTC
    if err := webrtc.SetupPeerConnection(client); err != nil {
        log.Printf("Failed to setup WebRTC: %v", err)
        return
    }

    room := getOrCreateRoom(roomID)
    room.AddClient(client)

    // Notify other clients about new peer
    room.BroadcastMessage(types.Message{
        Type:   "peer_joined",
        PeerID: peerID,
    }, peerID)

    // Send RSA public key
    initMsg := types.Message{
        Type:         "init",
        RSAPublicKey: crypto.GetPublicKeyPEM(),
        PeerID:       peerID,
        Room:         roomID,
    }
    if err := client.SendMessage(initMsg); err != nil {
        log.Printf("Failed to send init message: %v", err)
        return
    }

    // Message handling loop
    for {
        messageType, data, err := conn.ReadMessage()
        if err != nil {
            log.Printf("Read error from %s: %v", peerID, err)
            break
        }

        switch messageType {
        case websocket.TextMessage:
            if err := HandleJSONMessage(client, data, room); err != nil {
                log.Printf("Error handling JSON message from %s: %v", peerID, err)
                client.SendError(fmt.Sprintf("Message handling failed: %v", err))
            }

        case websocket.BinaryMessage:
            if err := HandleBinaryMessage(client, data); err != nil {
                log.Printf("Error handling binary message from %s: %v", peerID, err)
                client.SendError(fmt.Sprintf("Binary message handling failed: %v", err))
            }

        default:
            log.Printf("Unknown message type from %s: %d", peerID, messageType)
        }
    }

    // Cleanup
    client.Close()
    room.RemoveClient(peerID)
    
    // Notify other clients about peer leaving
    room.BroadcastMessage(types.Message{
        Type:   "peer_left",
        PeerID: peerID,
    }, peerID)
    
    log.Printf("Client %s disconnected from room %s", peerID, roomID)
}

func getOrCreateRoom(roomID string) *Room {
    roomsMutex.Lock()
    defer roomsMutex.Unlock()
    
    room, exists := rooms[roomID]
    if !exists {
        room = NewRoom()
        rooms[roomID] = room
    }
    return room
}