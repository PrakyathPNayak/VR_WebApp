package websocket

import (
    "log"
    "sync"
    "fmt"
    "VR-Distributed/pkg/types"
)

type Room struct {
    clients map[string]*Client
    mutex   sync.RWMutex
}

func NewRoom() *Room {
    return &Room{
        clients: make(map[string]*Client),
    }
}

func (r *Room) AddClient(client *Client) {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    r.clients[client.GetPeerID()] = client
}

func (r *Room) RemoveClient(peerID string) {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    delete(r.clients, peerID)
}

func (r *Room) BroadcastMessage(msg types.Message, excludePeerID string) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    
    for peerID, client := range r.clients {
        if peerID != excludePeerID {
            if err := client.SendMessage(msg); err != nil {
                log.Printf("Failed to send broadcast message to %s: %v", peerID, err)
            }
        }
    }
}

func (r *Room) ForwardMessage(msg types.Message, targetPeerID string) error {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    
    target, exists := r.clients[targetPeerID]
    if !exists {
        return fmt.Errorf("target peer %s not found", targetPeerID)
    }
    
    return target.SendMessage(msg)
}

func (r *Room) GetClientCount() int {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    return len(r.clients)
}