package webrtc

import (
    "fmt"
    "log"
    "time"

    "github.com/pion/webrtc/v3"
    "VR-Distributed/pkg/types"
)

type PeerInterface interface {
    GetPeerConnection() *webrtc.PeerConnection
    SetPeerConnection(*webrtc.PeerConnection)
    GetVideoTrack() *webrtc.TrackLocalStaticSample
    SetVideoTrack(*webrtc.TrackLocalStaticSample)
    GetAudioTrack() *webrtc.TrackLocalStaticSample
    SetAudioTrack(*webrtc.TrackLocalStaticSample)
    SendMessage(types.Message) error
    GetPeerID() string
}

// Global or package-level MediaEngine to register codecs once
var mediaEngine *webrtc.MediaEngine

func init() {
    mediaEngine = &webrtc.MediaEngine{}
    // Register default codecs. This sets up common codecs like H264 and Opus
    // with standard parameters within this mediaEngine instance.
    if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
        panic(fmt.Sprintf("Failed to register default codecs: %v", err))
    }
    // If you needed to register custom codecs or override defaults with specific SDP fmtp lines,
    // you would do it here using mediaEngine.RegisterCodec().
    // For example:
    if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
        RTPCodecCapability: webrtc.RTPCodecCapability{
            MimeType:    webrtc.MimeTypeH264,
            ClockRate:   90000,
            SDPFmtpLine: "profile-level-id=42e01f;packetization-mode=1",
        },
        PayloadType: 102, // This PayloadType is automatically assigned by RegisterDefaultCodecs for H264,
                          // but if you manually register, you might specify it.
    }, webrtc.RTPCodecTypeVideo); err != nil {
        panic(err)
    }
}

// GetAPI returns a new webrtc.API instance using the pre-configured mediaEngine.
func GetAPI() *webrtc.API {
    settingEngine := webrtc.SettingEngine{}
    // You can set various settings here if needed, like enabling ICE Lite:
    // settingEngine.SetLite(true)
    return webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithSettingEngine(settingEngine))
}

func SetupPeerConnection(client PeerInterface) error {
    config := webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {
                URLs: []string{"stun:stun.l.google.com:19302"},
            },
        },
    }

    peerConnection, err := GetAPI().NewPeerConnection(config)
    if err != nil {
        return fmt.Errorf("failed to create peer connection: %w", err)
    }

    client.SetPeerConnection(peerConnection)

    // --- Video Track Setup ---
    videoTrack, err := webrtc.NewTrackLocalStaticSample(
        webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
        "video",
        "stream",
    )
    if err != nil {
        return fmt.Errorf("failed to create video track: %w", err)
    }
    client.SetVideoTrack(videoTrack)

    videoTransceiver, err := peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
        Direction: webrtc.RTPTransceiverDirectionSendonly,
    })
    if err != nil {
        return fmt.Errorf("failed to add video transceiver: %w", err)
    }

    // Manually construct the H264 RTPCodecParameters.
    // When using RegisterDefaultCodecs, Pion handles the PayloadType assignment,
    // so you primarily need to ensure the MimeType and ClockRate match.
    // The PayloadType in this struct can be 0 or a placeholder, as Pion will
    // use the negotiated one.
    h264CodecParameters := webrtc.RTPCodecParameters{
        RTPCodecCapability: webrtc.RTPCodecCapability{
            MimeType: webrtc.MimeTypeH264,
            ClockRate: 90000, // Standard clock rate for H264
            // Channels and RTCPFeedback are typically not set for video here.
        },
        // PayloadType: 0, // PayloadType is usually determined during negotiation
    }

    // Set codec preferences for video (prioritize H264)
    // You can add more codecs here if you want to support fallback options.
    if err = videoTransceiver.SetCodecPreferences([]webrtc.RTPCodecParameters{h264CodecParameters}); err != nil {
        return fmt.Errorf("failed to set video codec preferences: %w", err)
    }

    // Add the video track to the transceiver's sender
    if err = videoTransceiver.Sender().ReplaceTrack(videoTrack); err != nil {
        return fmt.Errorf("failed to add video track to sender: %w", err)
    }


    // --- Audio Track Setup ---
    audioTrack, err := webrtc.NewTrackLocalStaticSample(
        webrtc.RTPCodecCapability{
            MimeType:  webrtc.MimeTypeOpus,
            ClockRate: 48000,
            Channels:  2,
        },
        "audio",
        "stream",
    )
    if err != nil {
        return fmt.Errorf("failed to create audio track: %w", err)
    }
    client.SetAudioTrack(audioTrack)

    audioTransceiver, err := peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{
        Direction: webrtc.RTPTransceiverDirectionSendonly,
    })
    if err != nil {
        return fmt.Errorf("failed to add audio transceiver: %w", err)
    }

    // Manually construct the Opus RTPCodecParameters.
    opusCodecParameters := webrtc.RTPCodecParameters{
        RTPCodecCapability: webrtc.RTPCodecCapability{
            MimeType:  webrtc.MimeTypeOpus,
            ClockRate: 48000, // Standard clock rate for Opus
            Channels:  2,     // Standard channels for Opus
        },
        // PayloadType: 0, // PayloadType is usually determined during negotiation
    }

    // Set codec preferences for audio (prioritize Opus)
    if err = audioTransceiver.SetCodecPreferences([]webrtc.RTPCodecParameters{opusCodecParameters}); err != nil {
        return fmt.Errorf("failed to set audio codec preferences: %w", err)
    }

    // Add the audio track to the transceiver's sender
    if err = audioTransceiver.Sender().ReplaceTrack(audioTrack); err != nil {
        return fmt.Errorf("failed to add audio track to sender: %w", err)
    }


    // Set up ICE candidate handling
    peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
        if candidate != nil {
            candidateInit := candidate.ToJSON()
            client.SendMessage(types.Message{
                Type:      "webrtc_ice_candidate",
                Candidate: &candidateInit,
                From:      client.GetPeerID(),
            })
        }
    })

    // Set up connection state change handling
    peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
        log.Printf("Peer connection state changed: %s", state.String())

        if state == webrtc.PeerConnectionStateConnected {
            client.SendMessage(types.Message{
                Type:    "status",
                Message: "WebRTC connection established",
            })
        } else if state == webrtc.PeerConnectionStateFailed {
            client.SendMessage(types.Message{
                Type:    "error",
                Message: "WebRTC connection failed",
            })
        }
    })

    return nil
}

func HandleOffer(client PeerInterface, msg types.Message) error {
    if msg.Offer == nil {
        return fmt.Errorf("no offer provided")
    }

    peerConnection := client.GetPeerConnection()

    // Set remote description
    if err := peerConnection.SetRemoteDescription(*msg.Offer); err != nil {
        return fmt.Errorf("failed to set remote description: %w", err)
    }

    // Create answer
    answer, err := peerConnection.CreateAnswer(nil)
    if err != nil {
        return fmt.Errorf("failed to create answer: %w", err)
    }

    // Set local description
    if err := peerConnection.SetLocalDescription(answer); err != nil {
        return fmt.Errorf("failed to set local description: %w", err)
    }

    // Send answer back
    answerMsg := types.Message{
        Type:   "answer",
        Answer: &answer,
        From:   client.GetPeerID(),
    }

    if err := client.SendMessage(answerMsg); err != nil {
        return fmt.Errorf("failed to send answer: %w", err)
    }

    // Start streaming media after connection establishes
    go func() {
        time.Sleep(2 * time.Second)
        // This would trigger media streaming - handled by media package
    }()

    return nil
}

func HandleAnswer(client PeerInterface, msg types.Message) error {
    if msg.Answer == nil {
        return fmt.Errorf("no answer provided")
    }

    peerConnection := client.GetPeerConnection()
    return peerConnection.SetRemoteDescription(*msg.Answer)
}

func HandleICECandidate(client PeerInterface, msg types.Message) error {
    if msg.Candidate == nil {
        return fmt.Errorf("no ICE candidate provided")
    }

    peerConnection := client.GetPeerConnection()
    return peerConnection.AddICECandidate(*msg.Candidate)
}