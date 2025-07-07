package media

import (
	"VR-Distributed/internal/webrtc"
	"encoding/binary"
	"fmt"
	"io"
	"log"
    "time"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	magicNumber = 0xDEADBEEF // Must match the VR process magic number
	headerSize  = 24         // Change if you use more fields
)

type FrameHeader struct {
    TimestampMS uint64
    Width       uint16
    Height      uint16
    FrameSize   uint32
}

type StreamerInterface interface {
	IsStreaming() bool
	SetStreaming(bool)
	GetStreamingMutex() *sync.RWMutex
	SendError(string)
	webrtc.MediaInterface
}

type VRProcess struct {
	Cmd    *exec.Cmd
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

func StartStreaming(client StreamerInterface, filePath string) error {
	log.Printf("Starting video streaming")
	if client.IsStreaming() {
		log.Printf("Already streaming")
		return fmt.Errorf("already streaming")
	}
	// client.SetStreaming(true)

	switch filepath.Ext(filePath) {
	case ".exe", ".elf": // find out how to do this for Linux
		go func() {
			defer func() {
				client.SetStreaming(false)
			}()
			// remove this if you want to stream video and audio separately
			if err := StartStreamingFromVR(client, filePath, "default"); err != nil {
				log.Printf("Error streaming VR: %v", err)
				client.SendError(fmt.Sprintf("Failed to stream VR: %v", err))
			}
		}()

	case ".mp4", ".mkv", ".webp": // add more if you want to
		// Start Video and Audio streaming
		go func() {
			defer func() {
				client.SetStreaming(false)
			}()
			if err := StreamVideoWithAudio(client, filePath); err != nil {
				log.Printf("Error streaming video or audio: %v", err)
				client.SendError(fmt.Sprintf("Failed to stream video or audio: %v", err))
			}
		}()
		// Start video streaming
		/*go func() {
		    defer func() {
		        client.GetStreamingMutex().Lock()
		        client.SetStreaming(false)
		        client.GetStreamingMutex().Unlock()
		    }()

		    if err := StreamVideoFile(client, mediaFile); err != nil {
		        log.Printf("Error streaming video: %v", err)
		        client.SendError(fmt.Sprintf("Failed to stream video: %v", err))
		    }
		}()*/

	case ".mp3", ".flac", ".wav", ".aac": // Add more if you want to
		// Start audio streaming
		go func() {
			defer func() {
				client.SetStreaming(false)			}()
			// remove this if you want to stream video and audio separately
			if err := StreamAudioFile(client, filePath); err != nil {
				log.Printf("Error streaming audio: %v", err)
				client.SendError(fmt.Sprintf("Failed to stream audio: %v", err))
			}
		}()
	}
	log.Printf("Done start_vr")
	return nil
}

func StopStreaming(client StreamerInterface) {
	mutex := client.GetStreamingMutex()
	mutex.Lock()
	defer mutex.Unlock()

	client.SetStreaming(false)
}

func StartStreamingFromVR(client StreamerInterface, exePath, room string) error {
	log.Println("Starting VR streaming")

	if client.IsStreaming() {
		return fmt.Errorf("already streaming")
	}

	client.SetStreaming(true)

	go func() {
		defer func() {
			client.GetStreamingMutex().Lock()
			client.SetStreaming(false)
			client.GetStreamingMutex().Unlock()
		}()

		vr, err := StartVRProcess(exePath, room)
		if err != nil {
			log.Printf("Failed to start VR process: %v", err)
			client.SendError(fmt.Sprintf("Failed to start VR process: %v", err))
			return
		}
		defer vr.Cmd.Process.Kill() // Clean up

		if err := StreamVRVideo(client, vr); err != nil {
			log.Printf("Error streaming VR video: %v", err)
			client.SendError(fmt.Sprintf("VR streaming error: %v", err))
		}
	}()

	return nil
}

func StartVRProcess(exePath, room string) (*VRProcess, error) {
	cmd := exec.Command(exePath, "--webrtc", "--room", room)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	log.Printf("[VRProcess] Started process: %s", exePath)

	// Optional: Log stderr to debug issues
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				break
			}
			log.Printf("[VRProcess STDERR] %s", string(buf[:n]))
		}
	}()

	return &VRProcess{Cmd: cmd, Stdout: stdout, Stderr: stderr}, nil
}

// FrameHeaderSize must match exactly how many bytes your Python FrameHeader uses
const FrameHeaderSize = 8 + 2 + 2 + 4 // 16 bytes

var lastTimestamp uint64
var frameCount int
var lastLogTime = time.Now()

func StreamVRVideo(client StreamerInterface, vr *VRProcess) error {
    log.Println("Starting stream from VR process")
    r := vr.Stdout
    
    // Frame header structure: 8 bytes timestamp + 2 bytes width + 2 bytes height + 4 bytes frame size = 16 bytes
    const frameHeaderSize = 24
    
    // Pre-allocate buffers to avoid repeated allocations
    headerBuf := make([]byte, frameHeaderSize)
    var frameBuf []byte
    var rgbFrame []byte
    
    // FPS tracking variables
    var frameCount int
    lastLogTime := time.Now()
    var lastTimestamp uint64
    
    client.SetStreaming(true)
    log.Println("Streaming started")
    
    for client.IsStreaming() {
        // Read frame header
        _, err := io.ReadFull(r, headerBuf)
        if err != nil {
            if err == io.EOF {
                log.Println("Stream ended (EOF)")
                break
            }
            return fmt.Errorf("error reading header: %w", err)
        }
        
        // Parse header inline for better performance
        timestampMS := binary.LittleEndian.Uint64(headerBuf[0:8])
        width := binary.LittleEndian.Uint16(headerBuf[8:10])
        height := binary.LittleEndian.Uint16(headerBuf[10:12])
        frameSize := binary.LittleEndian.Uint32(headerBuf[12:16])
        
        // Skip empty or duplicate frames
        if frameSize == 0 || timestampMS == lastTimestamp {
            if frameSize > 0 {
                // Still need to read and discard duplicate frame data
                if cap(frameBuf) < int(frameSize) {
                    frameBuf = make([]byte, frameSize)
                } else {
                    frameBuf = frameBuf[:frameSize]
                }
                _, err = io.ReadFull(r, frameBuf)
                if err != nil {
                    return fmt.Errorf("error reading duplicate frame: %w", err)
                }
            }
            continue
        }
        lastTimestamp = timestampMS
        
        // Resize frame buffer if needed (avoid reallocation when possible)
        if cap(frameBuf) < int(frameSize) {
            frameBuf = make([]byte, frameSize)
        } else {
            frameBuf = frameBuf[:frameSize]
        }
        
        // Read frame data
        _, err = io.ReadFull(r, frameBuf)
        if err != nil {
            return fmt.Errorf("error reading frame: %w", err)
        }
        
        // Convert RGBA to RGB efficiently
        rgbSize := int(width) * int(height) * 3
        if cap(rgbFrame) < rgbSize {
            rgbFrame = make([]byte, rgbSize)
        } else {
            rgbFrame = rgbFrame[:rgbSize]
        }
        
        // Efficient RGBA to RGB conversion
        rgbIdx := 0
        for i := 0; i < len(frameBuf); i += 4 {
            rgbFrame[rgbIdx] = frameBuf[i]     // R
            rgbFrame[rgbIdx+1] = frameBuf[i+1] // G  
            rgbFrame[rgbIdx+2] = frameBuf[i+2] // B
            rgbIdx += 3
        }
        
        // Write raw frame to WebRTC
        err = webrtc.WriteVideoSample(client, rgbFrame)
        if err != nil {
            return fmt.Errorf("WebRTC write failed: %w", err)
        }
        
        // FPS logging (optimized)
        frameCount++
        now := time.Now()
        if now.Sub(lastLogTime) >= time.Second {
            fps := float64(frameCount) / now.Sub(lastLogTime).Seconds()
            log.Printf("Pipe FPS: %.2f", fps)
            frameCount = 0
            lastLogTime = now
        }
    }
    
    log.Println("Stream ended")
    return nil
}

// findAnnexBStartCode returns the index of the first Annex-B start code (0x000001 or 0x00000001) in buf, or -1 if not found.
func findAnnexBStartCode(buf []byte) int {
	for i := 0; i < len(buf)-3; i++ {
		if buf[i] == 0x00 && buf[i+1] == 0x00 {
			if buf[i+2] == 0x01 {
				return i
			} else if buf[i+2] == 0x00 && buf[i+3] == 0x01 {
				return i
			}
		}
	}
	return -1
}

func StreamAudioFile(client StreamerInterface, mediaFile string) error {
	log.Printf("Starting to stream audio file: %s", mediaFile)

	audioReader, cleanup, err := CreateAudioStream(mediaFile)
	if err != nil {
		return err
	}
	defer cleanup()

	buffer := make([]byte, 1024*32) // 4KB buffer for audio
	for client.IsStreaming() {
		n, err := audioReader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Printf("End of audio stream reached")
				break
			}
			return fmt.Errorf("error reading audio data: %w", err)
		}

		if n > 0 {
			if err := webrtc.WriteAudioSample(client, buffer[:n]); err != nil {
				return err
			}
		}
	}
	return nil
}

func StreamVideoWithAudio(client StreamerInterface, mediaFile string) error {
	log.Println("Starting the stream")
	videoReader, audioReader, cleanup, err := CreateMediaStreams(mediaFile)
	if err != nil {
		return err

	}
	defer cleanup()
	videoBuffer, audioBuffer := make([]byte, 1024*32), make([]byte, 1024*4)
	for client.IsStreaming() {
		log.Println("isStreaming was changed correctly")
		nv, errv := videoReader.Read(videoBuffer)
		log.Println("Reached videoReader")
		if errv != nil {
			if errv == io.EOF {
				log.Printf("End of Video stream reached")
				break
			}
			return fmt.Errorf("error reading video data: %w", errv)
		}
		if nv > 0 {
			if err := webrtc.WriteAudioSample(client, videoBuffer[:nv]); err != nil {
				return err
			}
		}
		log.Println("Reached audio part")
		na, erra := audioReader.Read(audioBuffer)
		log.Println("Reached audioReader")
		if erra != nil {
			if erra == io.EOF {
				log.Printf("End of Video stream reached")
				break
			}
			return fmt.Errorf("error reading video data: %w", erra)
		}
		if na > 0 {
			if err := webrtc.WriteAudioSample(client, audioBuffer[:na]); err != nil {
				return err
			}
		}
	}
	return nil
}

func StreamVideoFile(client StreamerInterface, mediaFile string) error {
	log.Printf("Starting StreamVideoFile")
	videoReader, cleanup, err := CreateVideoStream(mediaFile)
	if err != nil {
		return err
	}
	defer cleanup()

	buf := make([]byte, 0, 65536)
	tmp := make([]byte, 4096)

	for client.IsStreaming() {
		n, err := videoReader.Read(tmp)
		if err != nil {
			if err == io.EOF {
				log.Printf("End of video stream reached")
				break
			}
			return fmt.Errorf("error reading video data: %w", err)
		}
		buf = append(buf, tmp[:n]...)

		// Extract complete NALUs from buffer
		for {
			start := findAnnexBStartCode(buf)
			if start < 0 {
				break // No start code found, wait for more data
			}
			next := findAnnexBStartCode(buf[start+3:])
			if next < 0 {
				break // No second start code, wait for more data
			}
			nalu := buf[start : start+3+next]
			if len(nalu) > 3 {
				err := webrtc.WriteVideoSample(client, nalu)
				if err != nil {
					return err
				}
			}
			buf = buf[start+3+next:]
		}
		// Optionally, trim buffer if it grows too large
		if len(buf) > 2*65536 {
			buf = buf[len(buf)-65536:]
		}
	}
	return nil
}
