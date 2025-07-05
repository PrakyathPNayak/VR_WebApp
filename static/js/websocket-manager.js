/**
 * WebSocket connection management and message handling
 */

class WebSocketManager {
  constructor() {
    this.socket = null;
    this.isConnected = false;
    this.streamReady = false;
    this.vrStarted = false;
    this.roomName = "default";
    this.myPeerId = null;
  }

  connect() {
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    const wsUrl = `${protocol}://${window.location.host}/ws/webrtc/${this.roomName}/`;
    this.socket = new WebSocket(wsUrl);
    this.socket.binaryType = "arraybuffer";

    this.socket.onopen = () => {
      this.isConnected = true;
      if (window.uiManager) {
        window.uiManager.updateStatus(
          "Connected. Awaiting initialization...",
          "connected",
        );
      }
    };

    this.socket.onclose = () => {
      if (window.uiManager) {
        window.uiManager.updateStatus("Connection closed", "error");
      }
      this.isConnected = false;
      this.streamReady = false;
      this.vrStarted = false;
      if (window.uiManager) {
        window.uiManager.enableStartVrButton();
      }
      if (window.webrtcManager) {
        window.webrtcManager.peers.clear();
        window.uiManager.updatePeerList(window.webrtcManager.peers);
      }
    };

    this.socket.onerror = () => {
      if (window.uiManager) {
        window.uiManager.updateStatus("WebSocket error", "error");
      }
    };

    this.socket.onmessage = async (event) => {
      if (typeof event.data === "string") {
        const msg = JSON.parse(event.data);
        console.log("message Received" + event.data);
        await this.handleMessage(msg);
      } else {
        // Handle binary data for encrypted messages
        const decryptedMsg = await decryptMessage(event.data);
        if (decryptedMsg) {
          await this.handleMessage(decryptedMsg);
        }
      }
    };
  }

  async handleMessage(msg) {
    switch (msg.type) {
      case "init":
        this.myPeerId = msg.peer_id;
        this.roomName = msg.room;
        if (window.webrtcManager) {
          window.webrtcManager.setMyPeerId(this.myPeerId);
        }
        const keyExchangeData = await performKeyExchange(msg.rsa_public_key);
        this.sendMessage({
          type: "aes_key_exchange",
          encrypted_key: keyExchangeData.encrypted_key,
          iv: keyExchangeData.iv,
        });
        break;

      case "key_exchange_complete":
        if (window.uiManager) {
          window.uiManager.updateStatus(
            "Encryption established. Ready to start VR.",
            "connected",
          );
          window.uiManager.enableStartVrButton();
        }
        break;

      case "vr_ready":
        if (window.uiManager) {
          window.uiManager.updateStatus(
            "VR process started. Establishing WebRTC...",
            "webrtc",
          );
          window.uiManager.disableStartVrButton();
        }
        this.vrStarted = true;

        // Initiate WebRTC connection to the server itself
        if (window.webrtcManager) {
          await window.webrtcManager.createOffer(this.myPeerId);
        }
        break;

      case "status":
        if (window.uiManager) {
          window.uiManager.updateStatus(msg.message, "connected");
        }
        break;

      case "error":
        if (window.uiManager) {
          window.uiManager.updateStatus(`Error: ${msg.message}`, "error");
        }
        if (msg.message.includes("VR")) {
          if (window.uiManager) {
            window.uiManager.enableStartVrButton();
          }
          this.vrStarted = false;
        }
        break;

      // WebRTC signaling messages
      case "peer_joined":
        if (window.webrtcManager) {
          window.webrtcManager.handlePeerJoined(msg.peer_id, this.vrStarted);
        }
        break;

      case "peer_left":
        if (window.webrtcManager) {
          window.webrtcManager.handlePeerLeft(msg.peer_id);
        }
        break;

      case "webrtc_offer":
        if (window.webrtcManager) {
          await window.webrtcManager.handleOffer(msg.offer, msg.from);
        }
        break;

      case "answer":
        console.log("Received WebRTC answer:", msg.answer);
        if (window.webrtcManager) {
          await window.webrtcManager.handleAnswer(msg.answer, msg.from);
        }
        break;

      case "webrtc_ice_candidate":
        if (window.webrtcManager) {
          await window.webrtcManager.handleIceCandidate(
            msg.candidate,
            msg.from,
          );
        }
        break;

      case "vr_debugging_status":
        if (window.uiManager) {
          window.uiManager.updateVrDebuggingStatus(msg.enabled);
        }
        break;

      default:
        console.warn("Unknown message type:", msg.type);
        break;
    }
  }

  sendMessage(messageObj) {
    if (this.socket && this.isConnected) {
      this.socket.send(JSON.stringify(messageObj));
    }
  }

  async sendEncryptedMessage(messageObj) {
    if (!isEncryptionReady() || !this.isConnected) return;

    const encryptedData = await encryptMessage(messageObj);
    if (encryptedData && this.socket) {
      this.socket.send(encryptedData);
    }
  }

  startVR() {
    if (!this.isConnected || !isEncryptionReady()) {
      if (window.uiManager) {
        window.uiManager.updateStatus("Not ready to start VR", "error");
      }
      return;
    }

    if (window.uiManager) {
      window.uiManager.updateStatus("Starting VR process...", "connected");
      window.uiManager.disableStartVrButton();
    }
    this.sendEncryptedMessage({ type: "start_vr" });
  }

  sendControl(type) {
    this.sendEncryptedMessage({ type });
  }

  setQuality(value) {
    if (!isNaN(value)) {
      this.sendEncryptedMessage({ type: "quality", value });
    }
  }

  toggleVrDebugging(enabled) {
    this.sendEncryptedMessage({
      type: "toggle_vr_debugging",
      enabled: enabled,
    });
  }

  disconnect() {
    this.sendControl("terminate");
    // Don't close socket immediately to allow cleanup message
  }

  sendGyroData(alpha, beta, gamma, timestamp) {
    if (this.socket && this.isConnected) {
      this.sendMessage({
        type: "gyro",
        alpha,
        beta,
        gamma,
        timestamp,
      });
    }
  }
}

// Create global instance
window.websocketManager = new WebSocketManager();
