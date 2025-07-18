package media

import (
	"VR-Distributed/internal/shared"
	"VR-Distributed/internal/webrtc"
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
	"layeh.com/gopus"
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

var handWriter = &shared.SharedMemoryWriter{}
var isrunning bool = true

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
	AudioOut io.ReadCloser
	AudioErr io.ReadCloser
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
			log.Println("got to case")
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
		        client.SetStreaming(false)
		    }()

		    if err := StreamVideoFile(client, filePath); err != nil {
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
		vr, err := StartVRProcess(client, exePath, room)
		if err != nil {
			client.SendError(fmt.Sprintf("Failed to start VR process: %v", err))
			return
		}
		defer vr.Cmd.Process.Kill()

		if err := StreamVRVideo(client, vr); err != nil {
			client.SendError(fmt.Sprintf("VR streaming error: %v", err))
		}
	}()
	/*go func() {
		log.Println("Starting Mediapipe process")
		//start mediapipe process
		mediapipe, err := StartMediapipeProcess(room)
		if err != nil {
			client.SendError(fmt.Sprintf("Failed to start Mediapipe process: %v", err))
			log.Printf("Failed to start Mediapipe process: %v", err)
			return
		}
		defer mediapipe.Cmd.Process.Kill()
	}()*/

	return nil
}
func StartMediapipeProcess(room string) (*VRProcess, error) {
	dir, _ := os.Getwd()
	log.Printf("[MEDIAPIPE]Starting Mediapipe process in directory: %s", dir)
	mediapipe := exec.Command(".venv/Scripts/python.exe", "-u", "./execs/Mediapipe.py", "--room", room)
	//venv := filepath.Join(dir, ".venv", "Scripts")

	// mediapipe.Env = append(os.Environ(),
	// 	//fmt.Sprintf("PATH=%s;%s", venv, os.Getenv("PATH")),
	// 	fmt.Sprintf("VIRTUAL_ENV=%s", filepath.Join(dir, ".venv", "")),
	// 	fmt.Sprintf("PATH=%s", filepath.Join(dir, ".venv", "Scripts")),
	// )

	log.Printf("[MediapipeProcess] Process env: %s", mediapipe.Env)
	stdoutm, err := mediapipe.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderrm, err := mediapipe.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := mediapipe.Start(); err != nil {
		return nil, err
	}
	log.Printf("[MediapipeProcess] Started process: %s", mediapipe.Path)
	// Test Python executable first
	testCmd := exec.Command(".venv/Scripts/python.exe", "-m", "pip", "list")
	testCmd.Dir = dir
	if output, err := testCmd.CombinedOutput(); err != nil {
		log.Printf("[MEDIAPIPE ERROR] Python test failed: %v, output: %s", err, string(output))
		return nil, fmt.Errorf("python test failed: %v", err)
	} else {
		log.Printf("[MEDIAPIPE] Python test successful: %s", string(output))
	}

	go func() {
		log.Printf("[MediapipeProcess] Started reading")
		scanner := bufio.NewScanner(stdoutm)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("[MediapipeProcess STDOUT] %s\n", line)
			handWriter.WriteStdin([]byte(line+"\n"), isrunning, 1)
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Mediapipe stdout read error: %v", err)
		}
	}()
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
func StartVRProcess(client StreamerInterface, exePath, room string) (*VRProcess, error) {
	cmd := exec.Command(exePath, "--webrtc", "--room", room)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	log.Println("VR stdin pipe created")
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

	// 🎧 Start FFmpeg audio capture from VAC
	audioCmd := exec.Command("ffmpeg",
	    "-f", "dshow",
	    "-i", "audio=CABLE Output (VB-Audio Virtual Cable)",
	    "-ar", "48000",
	    "-ac", "2",
	    "-f", "s16le", // raw PCM
	    "-acodec", "pcm_s16le",
	    "-nostats",
	    "-loglevel", "quiet",
	    "pipe:1",
	)

	audioOut, err := audioCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	audioErr, err := audioCmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := audioCmd.Start(); err != nil {
		return nil, err
	}

	log.Println("[AudioCapture] FFmpeg started with VAC input")
	go func() {
	    buf := make([]byte, 1024)
	    for {
	        n, err := audioErr.Read(buf)
	        if err != nil {
	            break
	        }
	        log.Printf("[FFmpeg Audio STDERR] %s", string(buf[:n]))
	    }
	}()

	// Optional: Log VR process stderr
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

	return &VRProcess{
		Cmd:      cmd,
		Stdout:   stdout,
		Stderr:   stderr,
		AudioOut: audioOut,
		AudioErr: audioErr,
	}, nil
}

// FrameHeaderSize must match exactly how many bytes your Python FrameHeader uses

// var lastTimestamp uint64
// var frameCount int
// var lastLogTime = time.Now()

func StreamVRVideo(client StreamerInterface, vr *VRProcess) error {
	log.Println("Starting VR video and audio streaming")

	r := vr.Stdout
	a := vr.AudioOut
	headerBuf := make([]byte, headerSize)

	client.SetStreaming(true)

	// Start audio goroutine
	go func() {
	    const (
	        sampleRate    = 48000
	        channels      = 2
	        frameSize     = 960                             // 20ms at 48kHz
	        pcmBytes      = frameSize * channels * 2        // 2 bytes per int16 sample
	        maxDataBytes  = 1275                            // Opus maximum for one frame per packet
	    )

	    encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	    if err != nil {
	        log.Printf("Failed to create Opus encoder: %v", err)
	        client.SendError("Opus encoder init failed")
	        return
	    }

	    // Optional encoder tuning
	    encoder.SetBitrate(64000)
	    encoder.SetApplication(gopus.Audio)

	    rawBuf := make([]byte, pcmBytes)
	    pcmBuf := make([]int16, frameSize*channels)

	    for client.IsStreaming() {
	        _, err := io.ReadFull(a, rawBuf)
	        if err != nil {
	            if err != io.EOF {
	                log.Printf("Audio read failed: %v", err)
	            }
	            break
	        }

	        // PCM: little-endian bytes to int16
	        for i := 0; i < len(pcmBuf); i++ {
	            pcmBuf[i] = int16(binary.LittleEndian.Uint16(rawBuf[i*2:]))
	        }

	        // Encode using the correct maxDataBytes
	        encodedPkt, err := encoder.Encode(pcmBuf, frameSize, maxDataBytes)
	        if err != nil {
	            log.Printf("Opus encoding error: %v", err)
	            continue
	        }

	        err = webrtc.WriteAudioSample(client, encodedPkt, 20)
	        if err != nil {
	            log.Printf("Failed to write audio sample: %v", err)
	            break
	        }
	    }

	    log.Println("Audio stream ended")
	}()


	// Handle video stream in current goroutine
	var frameCount int
	lastLogTime := time.Now()

	for client.IsStreaming() {
		_, err := io.ReadFull(r, headerBuf)
		if err != nil {
			if err == io.EOF {
				log.Println("Video stream ended (EOF)")
				break
			}
			return fmt.Errorf("error reading video header: %w", err)
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

		frameBuf := make([]byte, header.FrameSize)
		_, err = io.ReadFull(r, frameBuf)
		if err != nil {
			return fmt.Errorf("error reading frame data: %w", err)
		}
		if header.PixelFormat == 2 {
			// Pass H.264 data directly to WebRTC
			err = webrtc.WriteVideoSample(client, frameBuf, 5) // 5ms was used because it gave ~50-60 fps video stream without any hiccups 
			if err != nil {
				return fmt.Errorf("WebRTC write failed: %w", err)
			}
		} else {
			log.Printf("Unsupported pixel format: %d", header.PixelFormat)
		}
		// FPS Logging
		frameCount++
		now := time.Now()
		if now.Sub(lastLogTime) >= time.Second {
			fps := frameCount
			log.Printf("Pipe FPS: %d", fps)
			frameCount = 0
			lastLogTime = now
		}
	}

	log.Println("Video stream ended")
	client.SetStreaming(false)
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
			if err := webrtc.WriteAudioSample(client, buffer[:n], 20); err != nil {
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
	if client.IsStreaming() {
		return fmt.Errorf("already streaming")
	}
	client.SetStreaming(true)

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
			if err := webrtc.WriteAudioSample(client, videoBuffer[:nv], 20); err != nil {
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
			if err := webrtc.WriteAudioSample(client, audioBuffer[:na], 20); err != nil {
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
