package media

import (
    "fmt"
    "io"
    "log"
    "os"
    "os/exec"
)

func CreateVideoStream(mediaFile string) (io.ReadCloser, func(), error) {
    // Check if file exists
    log.Printf("Checking if file exists")
    if _, err := os.Stat(mediaFile); os.IsNotExist(err) {
        log.Printf("File does not exist")
        return nil, nil, fmt.Errorf("media file does not exist: %s", mediaFile)
    }
    
    log.Printf("Creating video stream")
    // Use FFmpeg to read the file and output raw video data
    ffmpegCmd := exec.Command("ffmpeg",
        "-re",
        "-i", mediaFile,
        "-c:v", "libx264",
        "-preset", "veryfast",
        "-tune", "zerolatency",
        "-pix_fmt", "yuv420p",
        "-g", "30",
        "-keyint_min", "30",
        "-sc_threshold", "0",
        "-b:v", "4M",       // increased
        "-maxrate", "8M",
        "-bufsize", "10M",
        "-f", "h264",
        "pipe:1",
    )

    log.Printf("Running ffmpeg")
    // Get stdout pipe for video data
    videoOut, err := ffmpegCmd.StdoutPipe()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get FFmpeg stdout pipe: %w", err)
    }
    
    // Start FFmpeg
    if err := ffmpegCmd.Start(); err != nil {
        return nil, nil, fmt.Errorf("failed to start FFmpeg: %w", err)
    }
    
    // Create a cleanup function
    cleanup := func() {
        if err := ffmpegCmd.Process.Kill(); err != nil {
            log.Printf("Error killing FFmpeg process: %v", err)
        }
        if err := ffmpegCmd.Wait(); err != nil {
            log.Printf("FFmpeg finished with error: %v", err)
        }
    }
    return videoOut, cleanup, nil
}

func CreateAudioStream(mediaFile string) (io.ReadCloser, func(), error) {
    if _, err := os.Stat(mediaFile); os.IsNotExist(err) {
        return nil, nil, fmt.Errorf("media file does not exist: %s", mediaFile)
    }

    cmd := exec.Command("ffmpeg",
        "-re",
        "-i", mediaFile,
        "-vn",
        "-acodec", "pcm_s16le",
        "-ar", "48000",
        "-ac", "2",
        "-f", "s16le", // raw 16-bit PCM
        "pipe:1",
    )



    cmd.Stderr = os.Stderr

    audioOut, err := cmd.StdoutPipe()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return nil, nil, fmt.Errorf("failed to start ffmpeg: %w", err)
    }

    cleanup := func() {
        _ = cmd.Process.Kill()
        _ = cmd.Wait()
    }
    log.Printf("Finished encoding audio")
    return audioOut, cleanup, nil
}

func CreateMediaStreams(mediaFile string) (videoOut, audioOut io.ReadCloser, cleanup func(), err error) {
    // only works in Linux
    // We may need to use named pipes for this later
    if _, err := os.Stat(mediaFile); os.IsNotExist(err) {
        return nil, nil, nil, fmt.Errorf("media file does not exist: %s", mediaFile)
    }

    audioRead, audioWrite, err := os.Pipe()
    if err != nil {
        return nil, nil, nil, fmt.Errorf("failed to create audio pipe: %w", err)
    }

    cmd := exec.Command("ffmpeg",
        "-re",
        "-i", mediaFile,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        "-tune", "zerolatency",
        "-pix_fmt", "yuv420p",
        "-profile:v", "baseline",
        "-level", "3.1",
        "-g", "30",
        "-keyint_min", "30",
        "-sc_threshold", "0",
        "-b:v", "1M",
        "-maxrate", "1M",
        "-bufsize", "2M",
        "-f", "h264",
        "pipe:1", // stdout for video

        "-c:a", "libopus",
        "-ar", "48000",
        "-ac", "2",
        "-b:a", "128k",
        "-f", "opus",
        "pipe:2", // fd 3 in Go
    )

    // Wire pipe:2 (audio) as ExtraFile
    cmd.ExtraFiles = []*os.File{audioWrite}

    videoOut, err = cmd.StdoutPipe()
    if err != nil {
        return nil, nil, nil, fmt.Errorf("failed to get video stdout pipe: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return nil, nil, nil, fmt.Errorf("failed to start FFmpeg: %w", err)
    }

    // Close writer end in parent
    _ = audioWrite.Close()

    cleanup = func() {
        _ = cmd.Process.Kill()
        _ = cmd.Wait()
    }

    return videoOut, audioRead, cleanup, nil
}


func ValidateMediaFile(mediaFile string) error {
    if _, err := os.Stat(mediaFile); os.IsNotExist(err) {
        return fmt.Errorf("media file does not exist: %s", mediaFile)
    }
    
    // Check both video and audio streams
    ffprobeCmd := exec.Command("ffprobe",
        "-v", "error",
        "-show_entries", "stream=codec_type",
        "-of", "csv=p=0",
        mediaFile,
    )
    
    output, err := ffprobeCmd.Output()
    if err != nil {
        return fmt.Errorf("failed to probe media file: %w", err)
    }
    
    log.Printf("Media streams found: %s", string(output))
    return nil
}