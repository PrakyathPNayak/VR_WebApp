import {
  HandLandmarker,
  FilesetResolver,
} from "https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.0";

/**
 * Handles hand tracking using MediaPipe's HandLandmarker
 */
class HandTrackingManager {
  constructor() {
    this.handLandmarker = null;
    this.trackingInterval = null;
    this.videoElement = document.getElementById("handTrackingVideo");
    this.cameraSelect = document.getElementById("cameraSelect");
    this.startBtn = document.getElementById("startHandTrackingBtn");
  }

  async initialize() {
    await this.populateCameraOptions();
    await this.initializeMediaPipe();

    this.startBtn.onclick = () => this.startTracking();
  }

  async populateCameraOptions() {
    try {
      // Step 1: Request user permission with a temporary generic camera stream
      const tempStream = await navigator.mediaDevices.getUserMedia({
        video: true,
      });

      // Step 2: Stop the stream immediately after access is granted
      tempStream.getTracks().forEach((track) => track.stop());

      // Step 3: Now enumerate devices with full labels
      const devices = await navigator.mediaDevices.enumerateDevices();
      const videoDevices = devices.filter((d) => d.kind === "videoinput");

      // Step 4: Populate dropdown
      this.cameraSelect.innerHTML = "";
      videoDevices.forEach((device, index) => {
        const option = document.createElement("option");
        option.value = device.deviceId;
        option.textContent = device.label || `Camera ${index + 1}`;
        this.cameraSelect.appendChild(option);
      });

      console.log("[hand-tracking] Cameras listed:", videoDevices);
    } catch (error) {
      console.error(
        "Camera permission denied or error during enumeration",
        error,
      );
      alert("Camera access is required to select and use hand tracking.");
    }
  }

  async initializeMediaPipe() {
    const vision = await FilesetResolver.forVisionTasks(
      "https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.0/wasm",
    );

    this.handLandmarker = await HandLandmarker.createFromOptions(vision, {
      baseOptions: {
        modelAssetPath:
          "https://storage.googleapis.com/mediapipe-models/hand_landmarker/hand_landmarker/float16/1/hand_landmarker.task",
        delegate: "GPU",
      },
      runningMode: "VIDEO",
      numHands: 2,
    });

    console.log("[hand-tracking] MediaPipe HandLandmarker initialized.");
  }

  async startTracking() {
    const deviceId = this.cameraSelect.value;
    if (!deviceId) {
      alert("Please select a camera device first.");
      return;
    }

    // Stop existing tracking if any
    this.stopTracking();

    // Start selected camera
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { deviceId: { exact: deviceId } },
    });

    this.videoElement.srcObject = stream;

    this.videoElement.onloadeddata = () => {
      console.log("[hand-tracking] Camera stream loaded.");
      this.predictLoop();
    };
  }

  predictLoop() {
    const loop = () => {
      // Stop the loop if the tracking has been stopped
      if (!this.videoElement.srcObject) {
        return;
      }

      const now = performance.now();
      const results = this.handLandmarker.detectForVideo(
        this.videoElement,
        now,
      );

      if (results.landmarks && results.landmarks.length > 0) {
        // 1. Map the raw landmark data to an array of hand objects.
        // This is the array that will go inside the "payload" key.
        const handsArray = results.landmarks.map((landmarkList, index) => {
          const handednessInfo = results.handednesses[index][0];
          return {
            handedness: handednessInfo.categoryName,
            landmarks: landmarkList,
            confidence: handednessInfo.score,
          };
        });

        // 2. Create the final data object in the required format.
        const dataToSend = {
          type: "hand",
          payload: handsArray,
        };

        // 3. Send the formatted data.
        // Assuming websocketManager will handle JSON.stringify if needed.
        window.websocketManager.sendHanddata(dataToSend);
      }

      // Request the next animation frame to continue the loop.
      window.requestAnimationFrame(loop);
    };

    // Start the prediction loop.
    window.requestAnimationFrame(loop);
  }

  stopTracking() {
    if (this.trackingInterval) {
      clearInterval(this.trackingInterval);
      this.trackingInterval = null;
    }

    if (this.videoElement.srcObject) {
      this.videoElement.srcObject.getTracks().forEach((track) => track.stop());
      this.videoElement.srcObject = null;
    }
  }
}

// Create global instance
window.handTrackingManager = new HandTrackingManager();
