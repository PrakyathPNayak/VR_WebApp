package media

import (
    "fmt"
    "io"
    "log"
    "sync"
    "VR-Distributed/internal/webrtc"
)

type StreamerInterface interface {
    IsStreaming() bool
    SetStreaming(bool)
    GetStreamingMutex() *sync.RWMutex
    SendError(string)
    webrtc.MediaInterface
}

func StartStreaming(client StreamerInterface, mediaFile string) error {
    log.Printf("Starting video streaming")
    if client.IsStreaming() {
        log.Printf("Already streaming")
        return fmt.Errorf("already streaming")
    }
    client.SetStreaming(true)
    // Start Video and Audio streaming
    /*go func(){
        defer func() {
            client.GetStreamingMutex().Lock()
            client.SetStreaming(false)
            client.GetStreamingMutex().Unlock()
        }()
        if err := StreamVideoWithAudio(client, mediaFile); err != nil {
            log.Printf("Error streaming video or audio: %v", err)
            client.SendError(fmt.Sprintf("Failed to stream video or audio: %v", err))
        }
    }()*/
    // Start video streaming
    go func() {
        defer func() {
            client.GetStreamingMutex().Lock()
            client.SetStreaming(false)
            client.GetStreamingMutex().Unlock()
        }()
        
        if err := StreamVideoFile(client, mediaFile); err != nil {
            log.Printf("Error streaming video: %v", err)
            client.SendError(fmt.Sprintf("Failed to stream video: %v", err))
        }
    }()
    
    // Start audio streaming
    /*go func() {
        defer func() {
            client.GetStreamingMutex().Lock()
            client.SetStreaming(false)
            client.GetStreamingMutex().Unlock()
        }()
        // remove this if you want to stream video and audio separately
        if err := StreamAudioFile(client, mediaFile); err != nil {
            log.Printf("Error streaming audio: %v", err)
            client.SendError(fmt.Sprintf("Failed to stream audio: %v", err))
        }
    }() */
    log.Printf("Done start_vr")
    return nil
}

func StopStreaming(client StreamerInterface) {
    mutex := client.GetStreamingMutex()
    mutex.Lock()
    defer mutex.Unlock()
    
    client.SetStreaming(false)
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

func StreamVideoWithAudio(client StreamerInterface, mediaFile string) error{
    log.Println("Starting the stream")
    videoReader, audioReader, cleanup, err := CreateMediaStreams(mediaFile)
    if err != nil {
        return err

    }
    defer cleanup()
    videoBuffer, audioBuffer := make([]byte, 1024*32), make([]byte, 1024*4)
    for client.IsStreaming(){
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