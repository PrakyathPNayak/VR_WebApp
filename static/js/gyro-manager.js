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
    const userAgent = navigator.userAgent || navigator.vendor || window.opera;
    const isiOS = /iPad|iPhone|iPod/.test(userAgent) && !window.MSStream;
    const isAndroid = /Android/i.test(userAgent);

    console.log(
      "Setting up motion tracking (iOS:",
      isiOS,
      ", Android:",
      isAndroid,
      ")",
    );

    this._listener = (event) => {
      const now = Date.now();

      if (
        !window.websocketManager ||
        !window.websocketManager.isConnected ||
        now - this.lastSentTime < this.gyroInterval
      ) {
        return;
      }

      let { alpha, beta, gamma } = event;

      // Handle orientation from screen
      const orientation = screen.orientation?.angle || window.orientation || 0;

      if (isiOS) {
        // Normalize for iOS to act like Android
        switch (orientation) {
          case 0: // Portrait
            // Flip beta and gamma to match Android portrait
            beta = -beta;
            gamma = -gamma;
            break;
          case 90: // Landscape left (home button on right)
            [beta, gamma] = [-gamma, beta];
            break;
          case -90:
          case 270: // Landscape right (home button on left)
            [beta, gamma] = [gamma, -beta];
            break;
          case 180: // Upside-down portrait
            beta = beta;
            gamma = -gamma;
            break;
          default:
            console.warn("Unknown screen orientation:", orientation);
        }
      } else if (isAndroid) {
        // Apply only Android-specific normalization if needed
        if (orientation === 0) {
          beta = -beta;
          gamma = -gamma;
        } else if (
          orientation === 90 ||
          orientation === -90 ||
          orientation === 270
        ) {
          [beta, gamma] = [gamma, beta];
          gamma = -gamma;
        }
      }

      // Send data
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
