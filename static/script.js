
const canvas = document.getElementById("canvas");
const ctx = canvas.getContext("2d");
const statusDiv = document.getElementById("status");
const qualitySlider = document.getElementById("quality");
const qualityValue = document.getElementById("qualityValue");

let socket,
  aesKey,
  iv,
  isConnected = false,
  streamReady = false;

let lastSentTime = 0;
const gyroInterval = 50;

qualitySlider.oninput = () => {
  qualityValue.textContent = qualitySlider.value;
};

function updateStatus(msg, type = "") {
  statusDiv.textContent = msg;
  statusDiv.className = `status ${type}`;
}

function connect() {
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const wsUrl = `${protocol}://${window.location.host}/ws/stream/`;
  socket = new WebSocket(wsUrl);
  socket.binaryType = "arraybuffer";

  socket.onopen = () => {
    isConnected = true;
    updateStatus("Connected. Awaiting RSA key...", "connected");
  };

  socket.onclose = () => {
    updateStatus("Connection closed", "error");
    isConnected = false;
    streamReady = false;
  };

  socket.onerror = () => {
    updateStatus("WebSocket error", "error");
  };

  socket.onmessage = async (event) => {
    if (typeof event.data === "string") {
      const msg = JSON.parse(event.data);
      await handleMessage(msg);
    } else {
      await handleEncryptedFrame(event.data);
    }
  };
}

function base64Encode(buffer) {
  return btoa(String.fromCharCode(...buffer));
}

function base64Decode(str) {
  return new Uint8Array([...atob(str)].map((c) => c.charCodeAt(0)));
}

async function performKeyExchange(encodedPem) {
  const pem = atob(encodedPem);
  const encrypt = new JSEncrypt();
  encrypt.setPublicKey(pem);

  const aesKeyRaw = crypto.getRandomValues(new Uint8Array(32));
  iv = crypto.getRandomValues(new Uint8Array(12));

  aesKey = await crypto.subtle.importKey(
    "raw",
    aesKeyRaw,
    { name: "AES-GCM" },
    false,
    ["encrypt", "decrypt"]
  );

  const aesKeyB64 = btoa(String.fromCharCode(...aesKeyRaw));
  const encryptedKey = encrypt.encrypt(aesKeyB64);

  socket.send(
    JSON.stringify({
      type: "aes_key_exchange",
      encrypted_key: encryptedKey,
      iv: base64Encode(iv),
    })
  );
}

async function handleMessage(msg) {
  switch (msg.type) {
    case "rsa_public_key":
      await performKeyExchange(msg.key);
      break;
    case "stream_ready":
      updateStatus("Streaming video...", "connected");
      streamReady = true;
      break;
    case "status":
      updateStatus(msg.message, "connected");
      break;
    case "error":
      updateStatus(`Error: ${msg.message}`, "error");
      break;
  }
}

async function handleEncryptedFrame(data) {
  if (!aesKey || !streamReady) return;

  const headerSize = 16;
  const nonceSize = 12;
  const tagSize = 16;

  const header = new DataView(data, 0, headerSize);
  const timestamp = header.getFloat64(0, true);
  const sequence = header.getUint32(8, true);
  const totalSize = header.getUint32(12, true);

  if (data.byteLength !== headerSize + totalSize) return;

  const nonce = data.slice(headerSize, headerSize + nonceSize);
  const ciphertext = data.slice(
    headerSize + nonceSize,
    data.byteLength - tagSize
  );
  const tag = data.slice(data.byteLength - tagSize);

  const fullData = new Uint8Array(ciphertext.byteLength + tag.byteLength);
  fullData.set(new Uint8Array(ciphertext), 0);
  fullData.set(new Uint8Array(tag), ciphertext.byteLength);

  try {
    const decrypted = await crypto.subtle.decrypt(
      {
        name: "AES-GCM",
        iv: nonce,
        tagLength: 128,
      },
      aesKey,
      fullData
    );

    // --- JPEG rendering ---

    const blob = new Blob([decrypted], { type: "image/jpeg" });
    const img = new Image();
    img.onload = () => {
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
      URL.revokeObjectURL(img.src);
    };
    img.onerror = (e) => {
      console.warn("Failed to load image", e);
      URL.revokeObjectURL(img.src);
    };
    img.src = URL.createObjectURL(blob);

    /*
            // --- H264 rendering using WebCodecs API ---
            const decoder = new VideoDecoder({
                output: frame => {
                    ctx.drawImage(frame, 0, 0, canvas.width, canvas.height);
                    frame.close();
                },
                error: e => console.error(e)
            });
            decoder.configure({ codec: 'avc1.42E01E', width: 1280, height: 720 });
            decoder.decode(new EncodedVideoChunk({
                type: 'key',
                timestamp,
                data: new Uint8Array(decrypted)
            }));*/
  } catch (e) {
    console.warn("Decryption failed", e);
  }
}

// --- Add encrypted message helper ---
async function sendEncryptedMessage(messageObj) {
  if (!aesKey || !iv || !isConnected) return;

  const encoded = new TextEncoder().encode(JSON.stringify(messageObj));
  const nonce = crypto.getRandomValues(new Uint8Array(12));

  try {
    const encrypted = await crypto.subtle.encrypt(
      {
        name: "AES-GCM",
        iv: nonce,
        tagLength: 128,
      },
      aesKey,
      encoded
    );

    const nonceAndEncrypted = new Uint8Array(nonce.byteLength + encrypted.byteLength);
    nonceAndEncrypted.set(nonce, 0);
    nonceAndEncrypted.set(new Uint8Array(encrypted), nonce.byteLength);

    socket.send(nonceAndEncrypted.buffer);
  } catch (e) {
    console.error("Failed to encrypt message:", e);
  }
}

function sendControl(type) {
  sendEncryptedMessage({ type });
}

function setQuality() {
  const value = parseInt(qualitySlider.value);
  if (!isNaN(value)) {
    sendEncryptedMessage({ type: "quality", value });
  }
}

function disconnect() {
  sendEncryptedMessage({ type: "terminate" });
  socket.close();
}

const enableGyroBtn = document.getElementById("enableGyroBtn");

async function enableGyro() {
  if (
    typeof DeviceOrientationEvent !== "undefined" &&
    typeof DeviceOrientationEvent.requestPermission === "function"
  ) {
    // iOS 13+ requires user gesture
    enableGyroBtn.style.display = "inline-block";
    enableGyroBtn.textContent = "Tap to Enable Gyro (Debug)";
    enableGyroBtn.onclick = async () => {
      console.log("Permission button clicked");
      try {
        const response = await DeviceOrientationEvent.requestPermission();
        if (response !== "granted") {
          updateStatus("Gyroscope access denied", "error");
          return;
        }
        setupGyroListener();
        enableGyroBtn.style.display = "none";
        updateStatus("Gyroscope enabled", "connected");
      } catch (e) {
        console.error("Permission request failed", e);
        updateStatus("Gyroscope permission failed", "error");
      }
    };
  } else {
    // Non-iOS or permission not required
    setupGyroListener();
  }
}

function setupGyroListener() {
  console.log("Gyro listener setup");
  window.addEventListener(
    "deviceorientation",
    (event) => {
      const now = Date.now();
      if (!socket || !isConnected || now - lastSentTime < gyroInterval) return;

      const { alpha, beta, gamma } = event;

      console.log("Gyro event:", alpha, beta, gamma);

      socket.send(
        JSON.stringify({
          type: "gyro",
          alpha,
          beta,
          gamma,
          timestamp: now,
        })
      );

      lastSentTime = now;
    },
    true
  );
}

window.onload = () => {
  connect();
  enableGyro();
};

window.onbeforeunload = () => socket && socket.close();
