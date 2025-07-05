package main

import (
    "log"
    "VR-Distributed/internal/config"
    "VR-Distributed/internal/crypto"
    "VR-Distributed/internal/server"
    "VR-Distributed/internal/webrtc"
)

func main() {
    // Load configuration
    cfg := config.Load()
    
    // Initialize crypto
    if err := crypto.InitializeRSA(); err != nil {
        log.Fatal("Failed to initialize RSA keys:", err)
    }
    
    // Initialize WebRTC
    if err := webrtc.Initialize(); err != nil {
        log.Fatal("Failed to initialize WebRTC:", err)
    }
    
    // Start server
    srv := server.New(cfg)
    log.Printf("WebRTC Media Server started on %s", cfg.ServerAddress)
    log.Fatal(srv.Start())
}