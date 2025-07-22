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

var stdinWriter = &shared.SharedMemoryWriter{}
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
		err := stdinWriter.NewSharedMemoryWriter("gyro.dat", 65536) // initialize the gyroWriter on key exchange complete
		if err != nil {
			log.Fatal("Failed to initialize gyro shared memory:", err)
			return err
		}
		configStruct := config.Load()
		go media.StartStreaming(client, configStruct.DefaultFilePath)
		client.SendMessage(types.Message{
			Type:    "vr_ready",
			Message: "VR process started",
		})
		log.Printf("VR started for client %s", client.GetPeerID())
		return nil

	case "stop_stream":
		return handleStopStream(client)

	case "webrtc_offer":
		return handleWebRTCOffer(client, msg, room)

	case "webrtc_answer":
		return handleWebRTCAnswer(client, msg, room)

	case "webrtc_ice_candidate":
		return handleWebRTCICECandidate(client, msg, room)

	/*case "encrypted_data":
		return handleEncryptedData(client, msg, room)*/
	case "start_handtracking":
		log.Println("Hand tracking has been initialized")
		return nil

	case "gyro":
		return handleGyroData(client, msg)

	case "hand":
		return handleHandData(client, msg)

	case "pause":
		log.Printf("Received pause command from %s", client.GetPeerID())
		// return handleStopStream(client)
		client.SetPaused(true)
		return nil

	case "resume":
		log.Printf("Received resume command from %s", client.GetPeerID())
		// return handleStartStream(client, msg)
		client.SetPaused(false)
		return nil

	case "terminate":
		log.Printf("Received terminate command from %s", client.GetPeerID())
		client.SetStreaming(false)
		stdinWriter.Close()
		return fmt.Errorf("client requested termination")
	
	case "quality":
		if value := msg.Value; value > 0 && value <= 100 {
			log.Printf("Received quality change from %s: %d", client.GetPeerID(), int(value))
		}
		return nil

	case "toggle_vr_debugging":
		if enabled := msg.Enabled; enabled {
			log.Printf("VR debugging toggled by %s: %t", client.GetPeerID(), enabled)
			client.SendMessage(types.Message{
				Type:    "vr_debugging_status",
				Message: fmt.Sprintf("VR debugging %s", map[bool]string{true: "enabled", false: "disabled"}[enabled]),
			})
		}
		return nil

	default:
		log.Printf("Unhandled JSON message type from %s: %s", client.GetPeerID(), msg.Type)
		return nil
	}
}

func HandleBinaryMessage(client *Client, data []byte, room *Room) error {
	// Decrypt the binary data
	decryptedData, err := client.DecryptBinaryData(data)
	if err != nil {
		return fmt.Errorf("binary decryption failed: %w", err)
	}

	// Parse decrypted data as JSON
	/*var controlMsg map[string]interface{}
	if err := json.Unmarshal(decryptedData, &controlMsg); err != nil {
		return fmt.Errorf("invalid binary control message: %w", err)
	}

	msgType, ok := controlMsg["type"].(string)
	if !ok {
		return fmt.Errorf("invalid binary control message type")
	}

	return handleControlMessage(client, msgType, controlMsg)*/
	return HandleJSONMessage(client, decryptedData, room)
}

func handleAESKeyExchange(client *Client, msg types.Message) error {
	key, err := crypto.DecryptAESKey(msg.EncryptedKey)
	if err != nil {
		return err
	}

	if err := client.SetupAESCipher(key); err != nil {
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

func handleEncryptedData(client *Client, msg types.Message, room *Room) error {
	decryptedData, err := client.DecryptData(msg.Data)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	/*var controlMsg map[string]interface{}
	if err := json.Unmarshal(decryptedData, &controlMsg); err != nil {
		return fmt.Errorf("invalid control message: %w", err)
	}

	msgType, ok := controlMsg["type"].(string)
	if !ok {
		return fmt.Errorf("invalid control message type")
	}

	return handleControlMessage(client, msgType, controlMsg)*/

	return HandleJSONMessage(client, decryptedData, room)
}

func handleGyroData(client *Client, msg types.Message) error {
	data := map[string]interface{}{
		"alpha":     msg.Alpha,
		"beta":      msg.Beta,
		"gamma":     msg.Gamma,
		"timestamp": time.Now().UnixMilli(),
	}
	// log.Printf("Received gyro data from %s: %+v", client.GetPeerID(), data)
	if err := stdinWriter.WriteStdin(data, isrunning, 0); err != nil {
		log.Println("Error writing gyro data to Stdin:", err)
	}

	return nil
}

func handleHandData(client *Client, msg types.Message) error {
	// 1. If the payload has no hands, it's a valid state (no hands in view).
	if len(msg.Hands.Payload) == 0 {
		return nil
	}

	// 2. Log receipt of data for operational awareness.
	log.Printf("📡 Hand data received from peer %s, writing to process.", client.GetPeerID())

	// 3. Pass the hand data payload to the generic writer.
	// We pass `msg.Hands.Payload`, which is a slice of Hand structs.
	// `WriteStdin` will marshal this slice into a JSON array.
	// `WriteStdinHandData` will then wrap it in the final object.
	// The '1' indicates this is hand data.
	if err := stdinWriter.WriteStdin(msg.Hands.Payload, isrunning, 1); err != nil {
		log.Println("Error writing hand data to Stdin:", err)
		// We log the error but return nil to allow the server to continue,
		// matching the pattern of a fire-and-forget handler.
	}

	return nil
}

func handleEncryptedHandData(client *Client, handData map[string]interface{}) error {
	rawHands, ok := handData["hands"]
	if !ok {
		log.Printf("🔸 'hands' field missing from handData for %s", client.GetPeerID())
		return nil
	}

	hands, ok := rawHands.([]interface{})
	if !ok {
		log.Printf("🔸 Invalid 'hands' format from %s", client.GetPeerID())
		return nil
	}

	log.Printf("  Hand data received from peer %s:", client.GetPeerID())

	for handIndex, hand := range hands {
		landmarks, ok := hand.([]interface{})
		if !ok {
			log.Printf("    Hand %d has invalid landmark data", handIndex+1)
			continue
		}

		log.Printf("    Hand %d:", handIndex+1)

		for landmarkIndex, point := range landmarks {
			coords, ok := point.([]interface{})
			if !ok || len(coords) < 3 {
				log.Printf("    Landmark %d → incomplete or invalid point data", landmarkIndex)
				continue
			}

			x, xOk := coords[0].(float64)
			y, yOk := coords[1].(float64)
			z, zOk := coords[2].(float64)

			if xOk && yOk && zOk {
				log.Printf("    Landmark %2d → x: %.4f, y: %.4f, z: %.4f", landmarkIndex, x, y, z)
			} else {
				log.Printf("    Landmark %2d → invalid coordinate types", landmarkIndex)
			}
		}
	}

	return nil
}



func handleControlMessage(client *Client, msgType string, controlMsg map[string]interface{}) error {
	switch msgType {
	case "start_vr":
		err := stdinWriter.NewSharedMemoryWriter("gyro.dat", 65536) // initialize the gyroWriter on key exchange complete
		if err != nil {
			log.Fatal("Failed to initialize gyro shared memory:", err)
			return err
		}
		configStruct := config.Load()
		go media.StartStreaming(client, configStruct.DefaultFilePath)
		client.SendMessage(types.Message{
			Type:    "vr_ready",
			Message: "VR process started",
		})
		log.Printf("VR started for client %s", client.GetPeerID())
	
	case "hand":
		return handleEncryptedHandData(client, controlMsg)

	case "pause":
		log.Printf("Received pause command from %s", client.GetPeerID())
		return handleStopStream(client)

	case "resume":
		log.Printf("Received resume command from %s", client.GetPeerID())
		// return handleStartStream(client, controlMsg)

	case "terminate":
		log.Printf("Received terminate command from %s", client.GetPeerID())
		client.SetStreaming(false)
		stdinWriter.Close()
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
