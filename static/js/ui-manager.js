/**
 * UI management and event handling
 */

class UIManager {
  constructor() {
    this.elements = {
      status: document.getElementById("status"),
      videoElement: document.getElementById("videoElement"),
      audioElement: document.getElementById("audioElement"),
      qualitySlider: document.getElementById("quality"),
      qualityValue: document.getElementById("qualityValue"),
      startVrBtn: document.getElementById("startVrBtn"),
      pauseBtn: document.getElementById("pauseBtn"),
      resumeBtn: document.getElementById("resumeBtn"),
      disconnectBtn: document.getElementById("disconnectBtn"),
      fullscreenBtn: document.getElementById("fullscreenBtn"),
      vrDebugBtn: document.getElementById("vrDebugBtn"),
      applyQualityBtn: document.getElementById("applyQualityBtn"),
      peerList: document.getElementById("peerList"),
      peerItems: document.getElementById("peerItems"),
    };

    this.vrDebugging = false;
    this.initializeEventListeners();
  }

  initializeEventListeners() {
    // Quality slider
    this.elements.qualitySlider.oninput = () => {
      this.elements.qualityValue.textContent =
        this.elements.qualitySlider.value;
    };

    // Button event listeners
    this.elements.startVrBtn.onclick = () => {
      if (window.websocketManager) {
        window.websocketManager.startVR();
      }
    };

    this.elements.pauseBtn.onclick = () => {
      if (window.websocketManager) {
        window.websocketManager.sendControl("pause");
      }
    };

    this.elements.resumeBtn.onclick = () => {
      if (window.websocketManager) {
        window.websocketManager.sendControl("resume");
      }
    };

    this.elements.disconnectBtn.onclick = () => {
      if (window.webrtcManager) {
        window.webrtcManager.disconnect();
      }
      if (window.gyroManager) {
        window.gyroManager.disableGyro();
      }
      if (window.websocketManager) {
        window.websocketManager.disconnect();
      }
    };

    this.elements.applyQualityBtn.onclick = () => {
      const value = parseInt(this.elements.qualitySlider.value);
      if (window.websocketManager) {
        window.websocketManager.setQuality(value);
      }
    };

    this.elements.fullscreenBtn.onclick = async () => {
      try {
        if (screen.orientation && screen.orientation.lock) {
          await screen.orientation.lock("landscape");
        }
      } catch (e) {
        console.warn("Orientation lock failed:", e);
      }

      if (this.elements.videoElement.requestFullscreen) {
        this.elements.videoElement.requestFullscreen();
      } else if (this.elements.videoElement.webkitRequestFullscreen) {
        this.elements.videoElement.webkitRequestFullscreen();
      } else if (this.elements.videoElement.msRequestFullscreen) {
        this.elements.videoElement.msRequestFullscreen();
      }
    };

    this.elements.vrDebugBtn.onclick = () => {
      this.vrDebugging = !this.vrDebugging;
      if (window.websocketManager) {
        window.websocketManager.toggleVrDebugging(this.vrDebugging);
      }
      this.updateVrDebuggingStatus(this.vrDebugging);
      this.updateStatus(
        `VR Debugging ${this.vrDebugging ? "enabled" : "disabled"}.`,
        "connected",
      );
    };

    // Window resize handler
    window.addEventListener("resize", () => this.resizeVideo());

    // Before unload handler
    window.addEventListener("beforeunload", () => {
      if (window.webrtcManager) {
        window.webrtcManager.disconnect();
      }
    });
  }

  updateStatus(msg, type = "") {
    this.elements.status.textContent = msg;
    this.elements.status.className = `status ${type}`;
  }

  updatePeerList(peers) {
    if (peers.size === 0) {
      this.elements.peerList.style.display = "none";
      return;
    }

    this.elements.peerList.style.display = "block";
    this.elements.peerItems.innerHTML = "";
    peers.forEach((peer, peerId) => {
      const div = document.createElement("div");
      div.className = "peer-item";
      div.textContent = `Peer: ${peerId} (${peer.state || "unknown"})`;
      this.elements.peerItems.appendChild(div);
    });
  }

  updateVrDebuggingStatus(enabled) {
    this.vrDebugging = enabled;
    this.elements.vrDebugBtn.textContent = `VR Debugging: ${enabled ? "ON" : "OFF"}`;
    this.elements.vrDebugBtn.style.background = enabled ? "#ffc107" : "#28a745";
  }

  enableStartVrButton() {
    this.elements.startVrBtn.disabled = false;
  }

  disableStartVrButton() {
    this.elements.startVrBtn.disabled = true;
  }

  resizeVideo() {
    const container = document.querySelector(".container");
    if (container) {
      this.elements.videoElement.style.maxWidth = "100%";
      this.elements.videoElement.style.height = "auto";
    }
  }

  initialize() {
    this.resizeVideo();
  }
}

// Create global instance
window.uiManager = new UIManager();
