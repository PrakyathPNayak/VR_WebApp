package types

import (
    "github.com/pion/webrtc/v3"
)

type Message struct {
    Type         string                     `json:"type"`
    EncryptedKey string                     `json:"encrypted_key,omitempty"`
    IV           string                     `json:"iv,omitempty"`
    RSAPublicKey string                     `json:"rsa_public_key,omitempty"`
    PeerID       string                     `json:"peer_id,omitempty"`
    Room         string                     `json:"room,omitempty"`
    Data         string                     `json:"data,omitempty"`
    Timestamp    int64                      `json:"timestamp,omitempty"`
    Error        string                     `json:"error,omitempty"`
    Message      string                     `json:"message,omitempty"`
    Hands        HandTrackingData           `json:"hands,omitempty"`
    // WebRTC specific fields
    Offer        *webrtc.SessionDescription `json:"offer,omitempty"`
    Answer       *webrtc.SessionDescription `json:"answer,omitempty"`
    Candidate    *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
    From         string                     `json:"from,omitempty"`
    Target       string                     `json:"target,omitempty"`
    
    // Additional fields
    Alpha        float64 `json:"alpha,omitempty"`
    Beta         float64 `json:"beta,omitempty"`
    Gamma        float64 `json:"gamma,omitempty"`
    Enabled      bool    `json:"enabled,omitempty"`
    Value        int     `json:"value,omitempty"`
}

// Landmark represents a single 3D coordinate (x, y, z).
type Landmark struct {
    X float32 `json:"x"`
    Y float32 `json:"y"`
    Z float32 `json:"z"`
}

// Hand contains all information for a single detected hand.
type Hand struct {
    Handedness string     `json:"handedness"`
    Landmarks  []Landmark `json:"landmarks"`
    Confidence float32    `json:"confidence"`
}

// HandTrackingData is the top-level object received from the JavaScript client.
type HandTrackingData struct {
    Type    string `json:"type"`
    Payload []Hand `json:"payload"`
}