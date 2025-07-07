/**
 * Gyroscope and motion tracking management
 */

class GyroManager {
  constructor() {
    this.enableGyroBtn = document.getElementById("enableGyroBtn");
    this.lastSentTime = 0;
    this.gyroInterval = 50;
    this.isEnabled = false;
    this._listener = null;
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
            window.uiManager?.updateStatus(
              "Motion tracking access denied",
              "error",
            );
            return;
          }
          this.setupGyroListener();
          this.enableGyroBtn.style.display = "none";
          window.uiManager?.updateStatus(
            "Motion tracking enabled",
            "connected",
          );
        } catch (e) {
          console.error("Permission request failed", e);
          window.uiManager?.updateStatus(
            "Motion tracking permission failed",
            "error",
          );
        }
      };
    } else {
      this.setupGyroListener();
    }
  }

  setupGyroListener() {
    const isAndroid = /Android/i.test(navigator.userAgent);
    console.log("Setting up motion tracking. Android:", isAndroid);

    this._listener = (event) => {
      const now = Date.now();
      if (
        !window.websocketManager ||
        !window.websocketManager.isConnected ||
        now - this.lastSentTime < this.gyroInterval
      )
        return;

      let { alpha, beta, gamma } = event;
      const orientation = screen.orientation?.angle || 0;

      if (isAndroid) {
        if (orientation === 0) {
          beta = -beta;
          gamma = -gamma;
        } else if (orientation === 90 || orientation === -90) {
          [beta, gamma] = [gamma, beta];
          gamma = -gamma;
        }
      }

      window.websocketManager.sendGyroData(alpha, beta, gamma, now);
      this.lastSentTime = now;
    };

    window.addEventListener("deviceorientation", this._listener, true);
    this.isEnabled = true;
  }

  disableGyro() {
    if (this._listener) {
      window.removeEventListener("deviceorientation", this._listener, true);
      this._listener = null;
    }
    this.isEnabled = false;
    console.log("Gyroscope tracking disabled");
  }

  isGyroEnabled() {
    return this.isEnabled;
  }
}

// Create global instance
window.gyroManager = new GyroManager();
