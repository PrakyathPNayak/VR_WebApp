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
    
    // WebRTC specific fields
    Offer     *webrtc.SessionDescription `json:"offer,omitempty"`
    Answer    *webrtc.SessionDescription `json:"answer,omitempty"`
    Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
    From      string                     `json:"from,omitempty"`
    Target    string                     `json:"target,omitempty"`
    
    // Additional fields
    Alpha     float64 `json:"alpha,omitempty"`
    Beta      float64 `json:"beta,omitempty"`
    Gamma     float64 `json:"gamma,omitempty"`
    Enabled   bool    `json:"enabled,omitempty"`
    Value     int     `json:"value,omitempty"`
}