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
    if (!this.handLandmarker || this.trackingInterval) return;

    this.trackingInterval = setInterval(() => {
      const now = performance.now();
      const results = this.handLandmarker.detectForVideo(
        this.videoElement,
        now,
      );

      if (results?.landmarks?.length > 0) {
        const handsData = results.landmarks.map((hand) =>
          hand.map(({ x, y, z }) => [x, y, z]),
        );

        window.websocketManager?.sendEncrytpedMessage({
          type: "hand",
          hands: handsData,
        });
      }
    }, 50); // ~20 fps
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
