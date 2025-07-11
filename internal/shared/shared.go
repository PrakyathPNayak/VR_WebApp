package shared

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/edsrzf/mmap-go"
)

type SharedMemoryWriter struct {
	file *os.File
	data mmap.MMap
	size int
}

var gyro_stdin io.WriteCloser
var mediapipe_stdin io.WriteCloser

func InitGyroStdin(stdin io.WriteCloser) {
	gyro_stdin = stdin
}
func InitmediapipeStdin(stdin io.WriteCloser) {
	mediapipe_stdin = stdin
}

func WriteStdinGyroData(json []byte, isrunning bool) error {
	if gyro_stdin == nil {
		return fmt.Errorf("gyro_stdin is not initialized")
	}
	json = append(json, '\n') // Ensure newline for proper parsing
	_, err := gyro_stdin.Write(json)
	if err != nil {
		log.Printf("Failed to write gyro data to stdin: %v", err)
		return fmt.Errorf("failed to write gyro data to stdin: %w", err)
	}
	//log.Printf("Wrote gyro data to stdin: %s", json)
	// if !isrunning {
	// 	log.Println("Gyro data writing is not running, closing gyro_stdin")
	// 	// Ensure the data is flushed to the VR process
	// 	if err := gyro_stdin.Close(); err != nil {
	// 		return fmt.Errorf("failed to close gyro_stdin: %w", err)
	// 	}
	// }
	return nil
}

func WriteStdinHandData(json []byte, isrunning bool) error {
	if mediapipe_stdin == nil {
		return fmt.Errorf("mediapipe_stdin is not initialized")
	}
	json = append(json, '\n') // Ensure newline for proper parsing
	_, err := mediapipe_stdin.Write(json)
	if err != nil {
		log.Printf("Failed to write gyro data to stdin: %v", err)
		return fmt.Errorf("failed to write gyro data to stdin: %w", err)
	}
	//log.Printf("Wrote hand data to stdin: %s", json)
	// if !isrunning {
	// 	log.Println("hand data writing is not running, closing mediapipe_stdin")
	// 	// Ensure the data is flushed to the VR process
	// 	if err := mediapipe_stdin.Close(); err != nil {
	// 		return fmt.Errorf("failed to close hand_stdin: %w", err)
	// 	}
	// }
	return nil
}

func (sharedPointer *SharedMemoryWriter) NewSharedMemoryWriter(filename string, size int) error {
	basePath, _ := os.Getwd()
	fullPath := filepath.Join(basePath, "Shared", filename)

	// Ensure file exists and has the desired size
	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := file.Truncate(int64(size)); err != nil {
		file.Close()
		return err
	}

	// Memory-map the file
	data, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return err
	}
	*sharedPointer = SharedMemoryWriter{
		file: file,
		data: data,
		size: size,
	}
	return nil
}

func (s *SharedMemoryWriter) WriteJSON(obj interface{}) error {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if len(jsonData) >= s.size {
		return os.ErrInvalid
	}

	copy(s.data, jsonData)
	s.data[len(jsonData)] = 0 // null-terminate
	return nil
}
func (s *SharedMemoryWriter) WriteStdin(obj interface{}, isrunning bool, datatype int) error {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if len(jsonData) >= s.size {
		return fmt.Errorf("JSON data too large for buffer: %d >= %d", len(jsonData), s.size)
	}
	jsonData = append(jsonData, '\n') // Ensure newline for proper parsing
	switch datatype {
	case 0: //case 0 for gyro
		if err := WriteStdinGyroData(jsonData, isrunning); err != nil {
			return fmt.Errorf("failed to write gyro data to stdin: %w", err)
		}
		return nil

	case 1: //case 1 for hand data
		if err := WriteStdinHandData(jsonData, isrunning); err != nil {
			return fmt.Errorf("failed to write hand data to stdin: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown datatype: %d", datatype)
	}
}

func (s *SharedMemoryWriter) Close() error {
	if err := s.data.Unmap(); err != nil {
		return err
	}
	err := s.file.Close()
	os.Remove(s.file.Name())
	return err
}
