package media

import (
    "fmt"
    "log"
    "sync"
    
    "webrtc-media-server/internal/webrtc"
)

type StreamerInterface interface {
    IsStreaming() bool
    SetStreaming(bool)
    GetStreamingMutex() *sync.RWMutex
    SendError(string)
    webrtc.MediaInterface
}

func StartStreaming(client StreamerInterface, mediaFile string) error {
    mutex := client.GetStreamingMutex()
    mutex.Lock()
    defer mutex.Unlock()
    
    if client.IsStreaming() {
        return fmt.Errorf("already streaming")
    }
    
    client.SetStreaming(true)
    
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
    go func() {
        if err := StreamAudioFile(client, mediaFile); err != nil {
            log.Printf("Error streaming audio: %v", err)
            client.SendError(fmt.Sprintf("Failed to stream audio: %v", err))
        }
    }()
    
    return nil
}

func StopStreaming(client StreamerInterface) {
    mutex := client.GetStreamingMutex()
    mutex.Lock()
    defer mutex.Unlock()
    
    client.SetStreaming(false)
}

func StreamVideoFile(client StreamerInterface, mediaFile string) error {
    log.Printf("Starting to stream video file: %s", mediaFile)
    
    videoReader, cleanup, err := CreateVideoStream(mediaFile)
    if err != nil {
        return err
    }
    defer cleanup()
    
    buffer := make([]byte, 1024*32) // 32KB buffer
    for client.IsStreaming() {
        n, err := videoReader.Read(buffer)
        if err != nil {
            if err == io.EOF {
                log.Printf("End of video stream reached")
                break
            }
            return fmt.Errorf("error reading video data: %w", err)
        }
        
        if n > 0 {
            if err := webrtc.WriteVideoSample(client, buffer[:n]); err != nil {
                return err
            }
        }
    }
    
    return nil
}

func StreamAudioFile(client StreamerInterface, mediaFile string) error {
    log.Printf("Starting to stream audio file: %s", mediaFile)
    
    audioReader, cleanup, err := CreateAudioStream(mediaFile)
    if err != nil {
        return err
    }
    defer cleanup()
    
    buffer := make([]byte, 1024*4) // 4KB buffer for audio
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