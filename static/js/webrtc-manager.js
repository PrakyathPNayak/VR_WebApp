/**
 * WebRTC connection management
 */

class WebRTCManager {
  constructor() {
    this.peers = new Map();
    this.localStream = null;
    this.myPeerId = null;
    this.videoElement = document.getElementById("videoElement");
    this.audioElement = document.getElementById("audioElement");
  }

  async createPeerConnection(peerId) {
    const config = {
      iceServers: [
        { urls: "stun:stun.l.google.com:19302" },
        { urls: "stun:stun1.l.google.com:19302" },
      ],
    };

    const pc = new RTCPeerConnection(config);

    // Ensure the offer includes a video media section
    if (
      !pc
        .getTransceivers()
        .some(
          (t) =>
            t.receiver && t.receiver.track && t.receiver.track.kind === "video",
        )
    ) {
      pc.addTransceiver("video", {
        direction: "recvonly",
        sendEncodings: [{ maxFramerate: 120 }],
      });
    }

    pc.addTransceiver("audio", { direction: "recvonly" });

    // Set maxBitrate for video sender (if sending video)
    pc.addEventListener("track", () => {
      setTimeout(() => {
        pc.getSenders().forEach((sender) => {
          if (sender.track && sender.track.kind === "video") {
            const params = sender.getParameters();
            if (!params.encodings) params.encodings = [{}];
            params.encodings[0].maxBitrate = 20_000_000; // 20 Mbps
            sender.setParameters(params);
            console.log("[WebRTC] Set maxBitrate to 20 Mbps for video sender");
          }
        });
      }, 0);
    });

    // Was used for debugging the WebRTC connection
    /*setInterval(() => {
      pc.getStats(null).then((stats) => {
        stats.forEach((report) => {
          if (report.type === "inbound-rtp") {
            console.log("🎥 Inbound RTP stats:", report);
          }
        });
      });
    }, 1000); */

    pc.onicecandidate = (event) => {
      if (event.candidate && event.candidate.candidate !== "") {
        console.log("Sending ICE candidate:", event.candidate);
        if (window.websocketManager) {
          window.websocketManager.sendMessage({
            type: "webrtc_ice_candidate",
            candidate: event.candidate,
            target: this.myPeerId,
          });
        }
      }
    };

    pc.oniceconnectionstatechange = () => {
      console.log("ICE State:", pc.iceConnectionState);
      const peer = this.peers.get(peerId);
      if (peer) {
        peer.state = pc.connectionState;
        if (window.uiManager) {
          window.uiManager.updatePeerList(this.peers);
        }
      }

      if (pc.connectionState === "connected") {
        if (window.uiManager) {
          window.uiManager.updateStatus(
            "WebRTC connection established",
            "webrtc",
          );
        }
      } else if (
        pc.connectionState === "failed" ||
        pc.connectionState === "disconnected"
      ) {
        if (window.uiManager) {
          window.uiManager.updateStatus("WebRTC connection lost", "error");
        }
      }
    };

    pc.ontrack = (event) => {
      console.log("Received remote stream (ontrack event):", event);

      const stream = event.streams[0];
      const trackKind = event.track.kind;

      if (trackKind === "video") {
        if (this.videoElement.srcObject !== stream) {
          this.videoElement.srcObject = stream;
          this.videoElement.autoplay = true;

          setTimeout(() => {
            this.videoElement.play().catch((err) => {
              if (err.name !== "AbortError") {
                console.error("Video playback error:", err);
              }
            });
          }, 100);
        }
      } else if (trackKind === "audio") {
        if (this.audioElement.srcObject !== stream) {
          this.audioElement.srcObject = stream;
          this.audioElement.autoplay = true;
          this.audioElement.muted = false;
          this.audioElement.volume = 1.0;

          setTimeout(() => {
            this.audioElement.play().catch((err) => {
              if (err.name !== "AbortError") {
                console.error("Audio playback error:", err);
              }
            });
          }, 100);
        }
      }

      console.log(
        `Track of kind '${trackKind}' attached to ${
          trackKind === "video" ? "videoElement" : "audioElement"
        }`,
      );

      if (window.uiManager) {
        window.uiManager.updateStatus("Receiving WebRTC stream", "webrtc");
      }
    };

    return pc;
  }

  async createOffer(peerId) {
    try {
      const pc = await this.createPeerConnection(peerId);
      this.peers.set(peerId, { pc, state: "connecting" });
      if (window.uiManager) {
        window.uiManager.updatePeerList(this.peers);
      }

      // Add local stream if available (for two-way communication)
      if (this.localStream) {
        this.localStream.getTracks().forEach((track) => {
          pc.addTrack(track, this.localStream);
        });
      }

      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      // Sanitize offer before sending
      if (!offer || !offer.sdp || !offer.type) {
        console.error("Generated invalid WebRTC offer:", offer);
        return;
      }
      console.log("Sending WebRTC offer:", offer);

      if (window.websocketManager) {
        window.websocketManager.sendMessage({
          type: "webrtc_offer",
          offer: offer,
          target: peerId,
        });
      }
    } catch (error) {
      console.error("Error creating offer:", error);
      if (window.uiManager) {
        window.uiManager.updateStatus("Failed to create WebRTC offer", "error");
      }
    }
  }

  async handleOffer(offer, fromPeer) {
    try {
      const pc = await this.createPeerConnection(fromPeer);
      this.peers.set(fromPeer, { pc, state: "connecting" });
      if (window.uiManager) {
        window.uiManager.updatePeerList(this.peers);
      }

      await pc.setRemoteDescription(offer);
      console.log("Received WebRTC offer:", offer);

      // Add local stream if available
      console.log("Adding local stream to PeerConnection for:", fromPeer);
      if (this.localStream) {
        this.localStream.getTracks().forEach((track) => {
          pc.addTrack(track, this.localStream);
        });
      }

      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      console.log("Sending WebRTC answer:", answer);

      if (window.websocketManager) {
        window.websocketManager.sendMessage({
          type: "webrtc_answer",
          answer: answer,
          target: fromPeer,
        });
      }
    } catch (error) {
      console.error("Error handling offer:", error);
      if (window.uiManager) {
        window.uiManager.updateStatus("Failed to handle WebRTC offer", "error");
      }
    }
  }

  async handleAnswer(answer, fromPeer) {
    const peer = this.peers.get(fromPeer);
    if (peer && peer.pc) {
      console.log("Received WebRTC answer:", answer);
      await peer.pc.setRemoteDescription(new RTCSessionDescription(answer));
    }
  }

  async handleIceCandidate(candidate, fromPeer) {
    const peer = this.peers.get(fromPeer);
    if (peer && peer.pc) {
      try {
        console.log("Adding ICE candidate:", candidate);
        await peer.pc.addIceCandidate(new RTCIceCandidate(candidate));
        console.log("Added ICE candidate:", candidate);
      } catch (err) {
        console.error("ICE candidate add failed", err, candidate);
      }
    }
  }

  handlePeerJoined(peerId, vrStarted) {
    console.log("Peer joined:", peerId);
    // Initiate connection to new peer if VR is started
    if (vrStarted && peerId !== this.myPeerId) {
      this.createOffer(peerId);
    }
  }

  handlePeerLeft(peerId) {
    console.log("Peer left:", peerId);
    const peer = this.peers.get(peerId);
    if (peer && peer.pc) {
      peer.pc.close();
      this.peers.delete(peerId);
      if (window.uiManager) {
        window.uiManager.updatePeerList(this.peers);
      }
    }
  }

  async initLocalStream() {
    try {
      const constraints = {
        video: {
          frameRate: { ideal: 60, max: 120 },
          width: { ideal: 1920 },
          height: { ideal: 1080 },
        },
        audio: true,
      };
      this.localStream = await navigator.mediaDevices.getUserMedia(constraints);
    } catch (error) {
      console.log("No local media stream needed or available");
    }
  }

  disconnect() {
    // Close all peer connections
    console.log("Closing all connections to WebRTC");
    this.peers.forEach((peer) => {
      if (peer.pc) {
        peer.pc.close();
      }
    });
    this.peers.clear();

    if (window.uiManager) {
      window.uiManager.updatePeerList(this.peers);
    }
    if (window.handTrackingManager) {
      window.handTrackingManager.stopTracking();
    }
    if (this.localStream) {
      this.localStream.getTracks().forEach((track) => track.stop());
      this.localStream = null;
    }
  }

  setMyPeerId(peerId) {
    this.myPeerId = peerId;
  }
}

// Create global instance
window.webrtcManager = new WebRTCManager();
