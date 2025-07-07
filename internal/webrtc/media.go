package webrtc

import (
    "io"
    "log"
    "sync"
    "time"
    "fmt"
    "github.com/pion/webrtc/v3"
    "github.com/pion/webrtc/v3/pkg/media"
)

type MediaInterface interface {
    GetVideoTrack() *webrtc.TrackLocalStaticSample
    GetAudioTrack() *webrtc.TrackLocalStaticSample
    IsStreaming() bool
    SetStreaming(bool)
    GetStreamingMutex() *sync.RWMutex
}

func WriteVideoSample(client MediaInterface, data []byte) error {
    if !client.IsStreaming() {
        return nil
    }
    
    videoTrack := client.GetVideoTrack()
    if videoTrack == nil {
        return fmt.Errorf("video track not available")
    }
    // log.Printf("Writing video track")
    sample := media.Sample{
        Data:     data,
        Duration: 33 * time.Millisecond, // ~30 FPS
    }
    
    if err := videoTrack.WriteSample(sample); err != nil {
        if err == io.ErrClosedPipe {
            log.Printf("Video track closed, stopping stream")
            client.SetStreaming(false)
            return nil
        }
        return fmt.Errorf("failed to write video sample: %w", err)
    }
    
    return nil
}

func WriteAudioSample(client MediaInterface, data []byte) error {
    if !client.IsStreaming() {
        return nil
    }
    audioTrack := client.GetAudioTrack()
    if audioTrack == nil {
        log.Printf("audio track not available")
        return fmt.Errorf("audio track not available")
    }
    
    sample := media.Sample{
        Data:     data,
        Duration: 20 * time.Millisecond, // 20ms audio frames
    }
    
    if err := audioTrack.WriteSample(sample); err != nil {
        if err == io.ErrClosedPipe {
            log.Printf("Audio track closed, stopping stream")
            client.SetStreaming(false)
            return nil
        }
        return fmt.Errorf("failed to write audio sample: %w", err)
    }
    
    return nil
}