# VR_WebApp

A secure, real-time VR web application powered by **Go** on the backend and **modern JavaScript** on the frontend. The system supports **end-to-end encrypted streaming**, **WebRTC-based video/audio communication**, **gyroscope motion tracking**, and **hand gesture recognition** using **MediaPipe** — optimized for VR control and remote streaming use cases.

---

## 🌐 Overview

The Go backend acts as the command center, managing:
- Encrypted WebSocket communication
- WebRTC signaling (SDP, ICE)
- RSA-AES key exchange
- Data routing for gyroscope, hand gestures, and control messages

The frontend is modularized for maintainability and handles:
- Stream playback and UI interaction
- MediaPipe-based hand tracking
- Secure encrypted message generation (AES-GCM)
- Motion tracking via device orientation sensors (gyroscope)

This project is especially suitable as a foundation for:
- VR-based remote streaming
- VR robot/telepresence control
- Real-time gesture-based control apps

---

## 🚀 Features

- ✅ **Secure RSA-AES Handshake** via Web Crypto API & JSEncrypt
- 🎥 **Low-latency WebRTC streaming** with real-time signaling over WebSocket
- 📡 **Encrypted sensor data** from gyroscope and hand tracking
- 🧠 **3D Hand landmark tracking** using MediaPipe Tasks
- 🌎 **Multi-peer session support** via room-based design
- 📲 **Mobile-compatible gyro tracking**, permission-managed for iOS/Android
- 🛠️ **Full modular frontend architecture** for easy extensibility

---

## 🛠 Technologies

| Stack Layer | Technology |
|-------------|------------|
| Backend     | Go (Golang), Gorilla WebSocket, WebRTC |
| Frontend    | JavaScript ES6 Modules, HTML5, CSS, Web Crypto API |
| Streaming   | WebRTC, MediaPipe Tasks Vision API |
| Encryption  | AES-GCM with 256-bit keys + RSA key transport |
| Sensors     | DeviceOrientationEvent, MediaPipe Hand Landmark Tracking |

---

## 📁 Project Structure

```
VR_WebApp/
├── static/
│   ├── js/
│   │   ├── crypto-utils.js       # RSA-AES key exchange & encryption utility
│   │   ├── websocket-manager.js  # WebSocket control + encrypted messaging
│   │   ├── webrtc-manager.js     # WebRTC signaling and stream handling
│   │   ├── gyro-manager.js       # Gyroscope tracking and data push
│   │   ├── hand-tracking.js      # MediaPipe-based hand gesture detection
│   │   ├── ui-manager.js         # Button bindings, sliders, event logic
│   │   └── main.js               # App bootstrapper & global wiring
│   ├── stylesheet.css
│   └── index.html
├── go/                            # Go server source files
│   ├── server.go
│   ├── client.go
│   ├── handlers.go
│   ├── types.go
│   └── message.go
└── README.md
```

---

## 🧪 Getting Started

### 1. Clone the repository

```
git clone https://github.com/PrakyathPNayak/VR_WebApp.git
cd VR_WebApp
```

### 2. Build & Run the Go Backend

```
cd go/
go build -o vrserver
./vrserver
```

> 🔐 Ensure you're serving over `localhost` or HTTPS for Web Crypto API compatibility.

### 3. Create and edit your env files

Though there are default options, I recommend the user to create their own config for security and compatibility purposes

You can find the header names in `config.go`

---

## 💡 Usage

- Click `Start VR` to begin encrypted handshake and VR session.
- Use the **"Pause"**, **"Resume"**, and **"Disconnect"** buttons to control session flow.
- Allow motion tracking + camera access when prompted.
- Use gesture/hand tracking by selecting a camera and clicking “Start Hand Tracking.”
- Stream quality can be adjusted with a slider.

---

## 📡 Message Types & Flow

| Message Type       | Handled by           | Purpose                          |
|--------------------|----------------------|----------------------------------|
| `init`             | Go Backend           | Start secure AES session         |
| `aes_key_exchange` | Go Backend           | Complete AES key exchange        |
| `vr_ready`         | Go Client → Server   | Starts gyroscope + WebRTC setup  |
| `gyro`             | Sent encrypted       | Streams device orientation data  |
| `hand_tracking`    | Sent encrypted       | Streams 3D hand landmark data    |
| `webrtc_offer`     | WebRTC peer signal   | Session Description Offer        |
| `answer`           | WebRTC peer signal   | Session Description Answer       |
| `candidate`        | ICE Negotiation      | NAT traversal info               |

---

## 🤝 Contributing

Pull requests, issues and feature suggestions are very welcome!

To contribute:

```
git checkout -b feature/my-feature
git commit -m "Added my feature"
git push origin feature/my-feature
```

Then submit a PR via GitHub.

---

## 📜 License

This project is developed by [PrakyathPNayak](https://github.com/PrakyathPNayak) and [Prajwal R.](https://github.com/Deadly-pro).

All code is currently released under the **Apache 2.0 License**.

See the [LICENSE](./LICENSE) file for details.

---

## 🙌 Credits

Special thanks to:

- MediaPipe Team for the open-source hand tracking models
- WebRTC and Web Crypto API communities
- Go open-source ecosystem
- Everyone contributing to privacy-first real-time tech

---

## 📬 Contact

Have questions or ideas?

[📧 prakyathpnayak@gmail.com](mailto:prakyathpnayak@gmail.com)  
[🌐 https://github.com/PrakyathPNayak](https://github.com/PrakyathPNayak)

---

**Experience VR, Securely. In Your Browser.**


- You can paste this directly into your `README.md`.
- Add a `LICENSE` file (MIT or Apache 2.0).
- Create a `go.mod` file (`go mod init vrwebapp`) if you haven’t.
- Want me to generate Dockerfiles or deployment instructions too? Just ask.

Let me know if you'd like the README in HTML format or want badges added (build status, MIT license, etc.).
