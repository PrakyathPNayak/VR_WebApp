package server

import (
	"log"
	"net/http"

	"VR-Distributed/internal/config"
	"VR-Distributed/internal/websocket"
)

type Server struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Start() error {
	// Serve static frontend files (stream.html, js, etc.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Serve the default video page
	http.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/stream.html")
	})

	// Register the WebSocket endpoint for WebRTC signaling
	http.HandleFunc("/ws/webrtc/", websocket.HandleWebRTCWS)

	// Start the HTTP server
	log.Printf("Starting server on %s", s.cfg.ServerAddress)
	return http.ListenAndServe(s.cfg.ServerAddress, nil)
}
