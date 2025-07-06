package server

import (
	"log"
	"net/http"
	"path/filepath"
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
	// Serve static frontend files with proper MIME types
	http.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Ext(r.URL.Path) {
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
	}))

	http.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/stream.html")
	})

	http.HandleFunc("/ws/webrtc/", websocket.HandleWebRTCWS)

	// Use HTTPS
	certPath := "cert.pem"
	keyPath := "key.pem"

	log.Printf("Starting HTTPS server on %s", s.cfg.ServerAddress)
	return http.ListenAndServeTLS(s.cfg.ServerAddress, certPath, keyPath, nil)
}
