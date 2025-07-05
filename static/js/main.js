/**
 * Main application entry point
 * Coordinates all modules and handles initialization
 */

class VRStreamApp {
  constructor() {
    this.initialized = false;
  }

  async initialize() {
    if (this.initialized) return;

    console.log("Initializing VR Stream Application...");

    try {
      // Initialize UI first
      if (window.uiManager) {
        window.uiManager.initialize();
      }

      // Initialize WebSocket connection
      if (window.websocketManager) {
        window.websocketManager.connect();
      }

      // Initialize gyroscope/motion tracking
      if (window.gyroManager) {
        await window.gyroManager.enableGyro();
      }

      // Initialize local media stream (optional - for two-way communication)
      // Uncomment if you need two-way communication
      // if (window.webrtcManager) {
      //   await window.webrtcManager.initLocalStream();
      // }

      this.initialized = true;
      console.log("VR Stream Application initialized successfully");
    } catch (error) {
      console.error("Failed to initialize VR Stream Application:", error);
      if (window.uiManager) {
        window.uiManager.updateStatus(
          "Failed to initialize application",
          "error",
        );
      }
    }
  }

  async shutdown() {
    console.log("Shutting down VR Stream Application...");

    try {
      // Disconnect WebRTC connections
      if (window.webrtcManager) {
        window.webrtcManager.disconnect();
      }

      // Disconnect WebSocket
      if (window.websocketManager) {
        window.websocketManager.disconnect();
      }

      this.initialized = false;
      console.log("VR Stream Application shutdown complete");
    } catch (error) {
      console.error("Error during application shutdown:", error);
    }
  }
}

// Create global app instance
window.vrStreamApp = new VRStreamApp();

// Initialize when DOM is loaded
window.addEventListener("DOMContentLoaded", async () => {
  await window.vrStreamApp.initialize();
});

// Handle page unload
window.addEventListener("beforeunload", () => {
  window.vrStreamApp.shutdown();
});

// For backwards compatibility, expose some global functions
window.startVR = () => {
  if (window.websocketManager) {
    window.websocketManager.startVR();
  }
};

window.sendControl = (type) => {
  if (window.websocketManager) {
    window.websocketManager.sendControl(type);
  }
};

window.setQuality = () => {
  const qualitySlider = document.getElementById("quality");
  const value = parseInt(qualitySlider.value);
  if (window.websocketManager) {
    window.websocketManager.setQuality(value);
  }
};

window.disconnect = () => {
  if (window.webrtcManager) {
    window.webrtcManager.disconnect();
  }
  if (window.websocketManager) {
    window.websocketManager.disconnect();
  }
};
