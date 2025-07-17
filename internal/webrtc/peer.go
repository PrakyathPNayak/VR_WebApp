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
    
    // Create video track
    videoTrack, err := webrtc.NewTrackLocalStaticSample(
        webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
        "video",
        "stream",
    )
    if err != nil {
        return fmt.Errorf("failed to create video track: %w", err)
    }
    client.SetVideoTrack(videoTrack)

    if _, err = peerConnection.AddTrack(videoTrack); err != nil {
        return fmt.Errorf("failed to add video track: %w", err)
    }
    // Uncomment if you want the audio track
    // Create audio track
    audioTrack, _ := webrtc.NewTrackLocalStaticSample(
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
    if _, err = peerConnection.AddTrack(audioTrack); err != nil {
        log.Println("Failed to add audio track")
        return fmt.Errorf("failed to add audio track: %w", err)
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