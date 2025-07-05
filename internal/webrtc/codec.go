package webrtc

import (
    "log"
    "github.com/pion/webrtc/v3"
)

var (
    mediaAPI *webrtc.MediaEngine
    api      *webrtc.API
)

func Initialize() error {
    mediaAPI = &webrtc.MediaEngine{}
    
    // Setup H264 codec
    if err := mediaAPI.RegisterCodec(webrtc.RTPCodecParameters{
        RTPCodecCapability: webrtc.RTPCodecCapability{
            MimeType:     webrtc.MimeTypeH264,
            ClockRate:    90000,
            Channels:     0,
            SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
            RTCPFeedback: nil,
        },
        PayloadType: 96,
    }, webrtc.RTPCodecTypeVideo); err != nil {
        return err
    }
    
    // Setup Opus codec
    if err := mediaAPI.RegisterCodec(webrtc.RTPCodecParameters{
        RTPCodecCapability: webrtc.RTPCodecCapability{
            MimeType:    webrtc.MimeTypeOpus,
            ClockRate:   48000,
            Channels:    2,
            SDPFmtpLine: "",
        },
        PayloadType: 111,
    }, webrtc.RTPCodecTypeAudio); err != nil {
        return err
    }
    
    api = webrtc.NewAPI(webrtc.WithMediaEngine(mediaAPI))
    log.Println("WebRTC codecs initialized successfully")
    return nil
}

func GetAPI() *webrtc.API {
    return api
}