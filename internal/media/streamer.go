package media

import (
	"VR-Distributed/internal/shared"
	"VR-Distributed/internal/webrtc"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const (
	magicNumber = 0xDEADBEEF // Must match the VR process magic number
	headerSize  = 24         // Change if you use more fields
)

// Match 6 uint32s = 24 bytes
type FrameHeader struct {
	Magic       uint32
	TimestampMS uint32
	FrameSize   uint32
	Width       uint32
	Height      uint32
	PixelFormat uint32
}

func parseHeader(data []byte) (*FrameHeader, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("invalid header length")
	}
	return &FrameHeader{
		Magic:       binary.LittleEndian.Uint32(data[0:4]),
		TimestampMS: binary.LittleEndian.Uint32(data[4:8]),
		FrameSize:   binary.LittleEndian.Uint32(data[8:12]),
		Width:       binary.LittleEndian.Uint32(data[12:16]),
		Height:      binary.LittleEndian.Uint32(data[16:20]),
		PixelFormat: binary.LittleEndian.Uint32(data[20:24]),
	}, nil
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
			// remove this if you want to stream video and audio separately.
			// Instead, call both audio and video adders one by one
			if err := StartStreamingFromVR(client, filePath, "default"); err != nil {
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
				client.SetStreaming(false)
			}()
			if err := StreamAudioFile(client, filePath); err != nil {
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
			client.SetStreaming(false)
		}()
		//start VR process
		vr, err := StartVRProcess(exePath, room)
		if err != nil {
			client.SendError(fmt.Sprintf("Failed to start VR process: %v", err))
			return
		}
		defer vr.Cmd.Process.Kill()

		if err := StreamVRVideo(client, vr); err != nil {
			client.SendError(fmt.Sprintf("VR streaming error: %v", err))
		}
	}()
	go func() {
		log.Println("Starting Mediapipe process")
		//start mediapipe process
		mediapipe, err := StartMediapipeProcess(room)
		if err != nil {
			client.SendError(fmt.Sprintf("Failed to start Mediapipe process: %v", err))
			log.Printf("Failed to start Mediapipe process: %v", err)
			return
		}
		defer mediapipe.Cmd.Process.Kill()
	}()

	return nil
}
func StartMediapipeProcess(room string) (*VRProcess, error) {
	//mediapipe exec
	mediapipe := exec.Command(".venv/Scripts/python.exe", "execs/Mediapipe.py", "--room", room)
	stdoutm, errm := mediapipe.StdoutPipe()
	if errm != nil {
		return nil, errm
	}
	if errm := mediapipe.Start(); errm != nil {
		return nil, errm
	}
	stderrm, err := mediapipe.StderrPipe()
	if err != nil {
		return nil, err
	}
	log.Printf("[MediapipeProcess] Started process: %s", mediapipe.Path)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutm.Read(buf)
			if err != nil {
				break
			}
			log.Printf("[MediapipeProcess STDOUT] %s", string(buf[:n]))
		}
	}()

	return &VRProcess{Cmd: mediapipe, Stdout: stdoutm, Stderr: stderrm}, nil

}
func StartVRProcess(exePath, room string) (*VRProcess, error) {
	cmd := exec.Command(exePath, "--webrtc", "--room", room)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	shared.InitGyroStdin(stdin) // Store the stdin for gyro data
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

// var lastTimestamp uint64
// var frameCount int
// var lastLogTime = time.Now()

func StreamVRVideo(client StreamerInterface, vr *VRProcess) error {
	log.Println("Starting stream from VR process")

	r := vr.Stdout
	headerBuf := make([]byte, headerSize)

	var lastTimestamp uint32
	client.SetStreaming(true)
	var frameCount int
	lastLogTime := time.Now()

	for client.IsStreaming() {
		_, err := io.ReadFull(r, headerBuf)
		if err != nil {
			if err == io.EOF {
				log.Println("Stream ended (EOF)")
				break
			}
			return fmt.Errorf("error reading header: %w", err)
		}

		header := FrameHeader{
			Magic:       binary.LittleEndian.Uint32(headerBuf[0:4]),
			TimestampMS: binary.LittleEndian.Uint32(headerBuf[4:8]),
			FrameSize:   binary.LittleEndian.Uint32(headerBuf[8:12]),
			Width:       binary.LittleEndian.Uint32(headerBuf[12:16]),
			Height:      binary.LittleEndian.Uint32(headerBuf[16:20]),
			PixelFormat: binary.LittleEndian.Uint32(headerBuf[20:24]),
		}

		if header.Magic != magicNumber {
			log.Printf("Invalid magic number: %x", header.Magic)
			continue
		}

		if header.FrameSize == 0 {
			log.Println("Skipping empty frame")
			continue
		}

		// Avoid duplicate timestamps
		/*if header.TimestampMS == lastTimestamp {
			log.Println("Skipping duplicate frame")
			// Consume the frame but skip sending it
			_, err = io.CopyN(io.Discard, r, int64(header.FrameSize))
			if err != nil {
				return fmt.Errorf("error skipping duplicate frame: %w", err)
			}
			continue
		}*/

		frameBuf := make([]byte, header.FrameSize)
		_, err = io.ReadFull(r, frameBuf)
		if err != nil {
			return fmt.Errorf("error reading frame data: %w", err)
		}
		currentTimestamp := header.TimestampMS
		duration := max(currentTimestamp-lastTimestamp, 7) // sets the fps to whatever 1000/7 is
		lastTimestamp = currentTimestamp
		if header.PixelFormat == 2 {
			// Pass H.264 data directly to WebRTC
			err = webrtc.WriteVideoSample(client, frameBuf, duration)
			if err != nil {
				return fmt.Errorf("WebRTC write failed: %w", err)
			}
		} else {
			log.Printf("Unsupported pixel format: %d", header.PixelFormat)
		}
		// FPS Logging
		frameCount++
		now := time.Now()
		if now.Sub(lastLogTime) >= time.Second && duration > 0 {
			fps := 1000 / duration
			log.Printf("Pipe FPS: %d", fps)
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
				err := webrtc.WriteVideoSample(client, nalu, 10)
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
