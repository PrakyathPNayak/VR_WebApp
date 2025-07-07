package websocket

import (
    "sync"
    "time"
    "fmt"
    "log"
    "github.com/gorilla/websocket"
    "github.com/pion/webrtc/v3"
    "VR-Distributed/internal/crypto"
    "VR-Distributed/pkg/types"
)

type Client struct {
    conn         *websocket.Conn
    peerID       string
    room         string
    lastPing     time.Time
    mutex        sync.RWMutex
    
    // Crypto
    aesCipher    *crypto.AESCipher
    
    // WebRTC
    peerConnection *webrtc.PeerConnection
    videoTrack     *webrtc.TrackLocalStaticSample
    audioTrack     *webrtc.TrackLocalStaticSample
    isStreaming    bool
    streamingMutex sync.RWMutex
}

func NewClient(conn *websocket.Conn, peerID, room string) *Client {
    return &Client{
        conn:     conn,
        peerID:   peerID,
        room:     room,
        lastPing: time.Now(),
    }
}

func (c *Client) SetupAESCipher(key []byte) error {
    cipher, err := crypto.NewAESCipher(key)
    if err != nil {
        return err
    }
    c.aesCipher = cipher
    return nil
}

func (c *Client) SendMessage(msg types.Message) error {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    msg.Timestamp = time.Now().UnixNano()
    return c.conn.WriteJSON(msg)
}

func (c *Client) SendError(errorMsg string) {
    msg := types.Message{
        Type:    "error",
        Message: errorMsg,
    }
    c.SendMessage(msg)
}

func (c *Client) GetPeerID() string {
    return c.peerID
}

func (c *Client) GetRoom() string {
    return c.room
}

func (c *Client) Close() error {
    if c.peerConnection != nil {
        c.peerConnection.Close()
    }
    return c.conn.Close()
}
func (c *Client) GetPeerConnection() *webrtc.PeerConnection {
    return c.peerConnection
}

func (c *Client) SetPeerConnection(pc *webrtc.PeerConnection) {
    c.peerConnection = pc
}

func (c *Client) GetVideoTrack() *webrtc.TrackLocalStaticSample {
    return c.videoTrack
}

func (c *Client) SetVideoTrack(track *webrtc.TrackLocalStaticSample) {
    c.videoTrack = track
}

func (c *Client) GetAudioTrack() *webrtc.TrackLocalStaticSample {
    return c.audioTrack
}

func (c *Client) SetAudioTrack(track *webrtc.TrackLocalStaticSample) {
    c.audioTrack = track
}

func (c *Client) IsStreaming() bool {
    c.streamingMutex.RLock()
    log.Println("check streaming")
    defer c.streamingMutex.RUnlock()
    return c.isStreaming
}

func (c *Client) SetStreaming(streaming bool) {
    c.streamingMutex.Lock()
    defer c.streamingMutex.Unlock()
    c.isStreaming = streaming
    log.Println("Set streaming")
}

func (c *Client) GetStreamingMutex() *sync.RWMutex {
    return &c.streamingMutex
}

func (c *Client) DecryptData(encryptedData string) ([]byte, error) {
    if c.aesCipher == nil {
        return nil, fmt.Errorf("decryption not initialized")
    }
    return c.aesCipher.Decrypt(encryptedData)
}

func (c *Client) DecryptBinaryData(data []byte) ([]byte, error) {
    if c.aesCipher == nil {
        return nil, fmt.Errorf("decryption not initialized")
    }
    return c.aesCipher.DecryptBinary(data)
}