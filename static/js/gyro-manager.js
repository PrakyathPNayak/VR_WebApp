/**
 * Gyroscope and motion tracking management
 */

class GyroManager {
  constructor() {
    this.enableGyroBtn = document.getElementById("enableGyroBtn");
    this.lastSentTime = 0;
    this.gyroInterval = 50; // milliseconds
    this.isEnabled = false;
  }

  async enableGyro() {
    if (
      typeof DeviceOrientationEvent !== "undefined" &&
      typeof DeviceOrientationEvent.requestPermission === "function"
    ) {
      this.enableGyroBtn.style.display = "inline-block";
      this.enableGyroBtn.textContent = "Tap to Enable Motion Tracking";
      this.enableGyroBtn.onclick = async () => {
        console.log("Permission button clicked");
        try {
          const response = await DeviceOrientationEvent.requestPermission();
          if (response !== "granted") {
            if (window.uiManager) {
              window.uiManager.updateStatus(
                "Motion tracking access denied",
                "error",
              );
            }
            return;
          }
          this.setupGyroListener();
          this.enableGyroBtn.style.display = "none";
          if (window.uiManager) {
            window.uiManager.updateStatus(
              "Motion tracking enabled",
              "connected",
            );
          }
        } catch (e) {
          console.error("Permission request failed", e);
          if (window.uiManager) {
            window.uiManager.updateStatus(
              "Motion tracking permission failed",
              "error",
            );
          }
        }
      };
    } else {
      this.setupGyroListener();
    }
  }

  setupGyroListener() {
    const isAndroid = /Android/i.test(navigator.userAgent);
    console.log("Setting up motion tracking. Android:", isAndroid);

    window.addEventListener(
      "deviceorientation",
      (event) => {
        const now = Date.now();
        if (
          !window.websocketManager ||
          !window.websocketManager.isConnected ||
          now - this.lastSentTime < this.gyroInterval
        ) {
          return;
        }

        let { alpha, beta, gamma } = event;
        const orientation = screen.orientation?.angle || 0;

        if (isAndroid) {
          if (orientation === 0) {
            // portrait
            beta = -beta;
            gamma = -gamma;
          } else if (orientation === 90 || orientation === -90) {
            // landscape
            // Swap and invert gamma
            [beta, gamma] = [gamma, beta];
            gamma = -gamma;
          }
        }

        if (window.websocketManager) {
          window.websocketManager.sendGyroData(alpha, beta, gamma, now);
        }

        this.lastSentTime = now;
      },
      true,
    );

    this.isEnabled = true;
  }

  isGyroEnabled() {
    return this.isEnabled;
  }
}

// Create global instance
window.gyroManager = new GyroManager();
