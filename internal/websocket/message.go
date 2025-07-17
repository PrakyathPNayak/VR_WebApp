package websocket

import (
	"VR-Distributed/internal/config"
	"VR-Distributed/internal/crypto"
	"VR-Distributed/internal/media"
	"VR-Distributed/internal/shared"
	"VR-Distributed/internal/webrtc"
	"VR-Distributed/pkg/types"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var gyroWriter = &shared.SharedMemoryWriter{}
var isrunning bool = true

func HandleJSONMessage(client *Client, data []byte, room *Room) error {
	var msg types.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	switch msg.Type {
	case "aes_key_exchange":
		return handleAESKeyExchange(client, msg)

	case "start_vr":
		return handleStartStream(client, msg)

	case "stop_stream":
		return handleStopStream(client)

	case "webrtc_offer":
		return handleWebRTCOffer(client, msg, room)

	case "webrtc_answer":
		return handleWebRTCAnswer(client, msg, room)

	case "webrtc_ice_candidate":
		return handleWebRTCICECandidate(client, msg, room)

	case "encrypted_data":
		return handleEncryptedData(client, msg)

	case "gyro":
		return handleGyroData(client, msg)

	default:
		log.Printf("Unhandled JSON message type from %s: %s", client.GetPeerID(), msg.Type)
		return nil
	}
}

func HandleBinaryMessage(client *Client, data []byte) error {
	// Decrypt the binary data
	decryptedData, err := client.DecryptBinaryData(data)
	if err != nil {
		return fmt.Errorf("binary decryption failed: %w", err)
	}

	// Parse decrypted data as JSON
	var controlMsg map[string]interface{}
	if err := json.Unmarshal(decryptedData, &controlMsg); err != nil {
		return fmt.Errorf("invalid binary control message: %w", err)
	}

	msgType, ok := controlMsg["type"].(string)
	if !ok {
		return fmt.Errorf("invalid binary control message type")
	}

	return handleControlMessage(client, msgType, controlMsg)
}

func handleAESKeyExchange(client *Client, msg types.Message) error {
	key, err := crypto.DecryptAESKey(msg.EncryptedKey)
	if err != nil {
		return err
	}

	if err := client.SetupAESCipher(key); err != nil {
		return err
	}

	err = gyroWriter.NewSharedMemoryWriter("gyro.dat", 65536) // initialize the gyroWriter on key exchange complete

	if err != nil {
		log.Fatal("Failed to initialize gyro shared memory:", err)
		return err
	}
	ack := types.Message{Type: "key_exchange_complete"}
	return client.SendMessage(ack)
}

func handleStartStream(client *Client, msg types.Message) error {
	mediaFile := msg.Data
	configStruct := config.Load()
	if mediaFile == "" {
		mediaFile = configStruct.DefaultFilePath // change it to whatever you want
	}

	go func() {
		if err := media.StartStreaming(client, mediaFile); err != nil {
			log.Printf("Failed to start media stream: %v", err)
			client.SendError(fmt.Sprintf("Failed to start stream: %v", err))
		}
	}()

	return client.SendMessage(types.Message{
		Type:    "stream_started",
		Message: fmt.Sprintf("Started streaming: %s", mediaFile),
	})
}

func handleStopStream(client *Client) error {
	media.StopStreaming(client)
	isrunning = false
	log.Println("Stream stopped for client:", client.GetPeerID(), "isrunning:", isrunning)
	return client.SendMessage(types.Message{
		Type:    "stream_stopped",
		Message: "Stream stopped",
	})
}

func handleWebRTCOffer(client *Client, msg types.Message, room *Room) error {
	if msg.Target != "" && msg.Target != client.GetPeerID() {
		return room.ForwardMessage(msg, msg.Target)
	}
	return webrtc.HandleOffer(client, msg)
}

func handleWebRTCAnswer(client *Client, msg types.Message, room *Room) error {
	if msg.Target != "" && msg.Target != client.GetPeerID() {
		return room.ForwardMessage(msg, msg.Target)
	}
	isrunning = true
	log.Println("Stream started for client:", client.GetPeerID(), "isrunning:", isrunning)
	return webrtc.HandleAnswer(client, msg)
}

func handleWebRTCICECandidate(client *Client, msg types.Message, room *Room) error {
	if msg.Target != "" && msg.Target != client.GetPeerID() {
		return room.ForwardMessage(msg, msg.Target)
	}
	return webrtc.HandleICECandidate(client, msg)
}

func handleEncryptedData(client *Client, msg types.Message) error {
	decryptedData, err := client.DecryptData(msg.Data)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	var controlMsg map[string]interface{}
	if err := json.Unmarshal(decryptedData, &controlMsg); err != nil {
		return fmt.Errorf("invalid control message: %w", err)
	}

	msgType, ok := controlMsg["type"].(string)
	if !ok {
		return fmt.Errorf("invalid control message type")
	}

	return handleControlMessage(client, msgType, controlMsg)
}

func handleGyroData(client *Client, msg types.Message) error {
	data := map[string]interface{}{
		"alpha":     msg.Alpha,
		"beta":      msg.Beta,
		"gamma":     msg.Gamma,
		"timestamp": time.Now().UnixMilli(),
	}
	//log.Printf("Received gyro data from %s: %+v", client.GetPeerID(), data)
	if err := gyroWriter.WriteStdin(data, isrunning, 0); err != nil {
		log.Println("Error writing gyro data to Stdin:", err)
	}

	return nil
}

func handleControlMessage(client *Client, msgType string, controlMsg map[string]interface{}) error {
	switch msgType {
	case "start_vr":
		configStruct := config.Load()
		go media.StartStreaming(client, configStruct.DefaultFilePath)
		client.SendMessage(types.Message{
			Type:    "vr_ready",
			Message: "VR process started",
		})
		log.Printf("VR started for client %s", client.GetPeerID())

	case "pause":
		log.Printf("Received pause command from %s", client.GetPeerID())

	case "resume":
		log.Printf("Received resume command from %s", client.GetPeerID())

	case "terminate":
		log.Printf("Received terminate command from %s", client.GetPeerID())
		client.SetStreaming(false)
		gyroWriter.Close()
		return fmt.Errorf("client requested termination")

	case "quality":
		if value, ok := controlMsg["value"].(float64); ok {
			log.Printf("Received quality change from %s: %d", client.GetPeerID(), int(value))
		}

	case "toggle_vr_debugging":
		if enabled, ok := controlMsg["enabled"].(bool); ok {
			log.Printf("VR debugging toggled by %s: %t", client.GetPeerID(), enabled)
			client.SendMessage(types.Message{
				Type:    "vr_debugging_status",
				Message: fmt.Sprintf("VR debugging %s", map[bool]string{true: "enabled", false: "disabled"}[enabled]),
			})
		}

	default:
		log.Printf("Unknown binary control message type from %s: %s", client.GetPeerID(), msgType)
	}

	return nil
}
