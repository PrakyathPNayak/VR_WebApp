package websocket

import (
	"VR-Distributed/internal/media"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/edsrzf/mmap-go"
)

type SharedMemoryWriter struct {
	file *os.File
	data mmap.MMap
	size int
}

func NewSharedMemoryWriter(filename string, size int) (*SharedMemoryWriter, error) {
	basePath, _ := os.Getwd()
	fullPath := filepath.Join(basePath, "Shared", filename)

	// Ensure file exists and has the desired size
	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	if err := file.Truncate(int64(size)); err != nil {
		file.Close()
		return nil, err
	}

	// Memory-map the file
	data, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &SharedMemoryWriter{
		file: file,
		data: data,
		size: size,
	}, nil
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

func (s *SharedMemoryWriter) WriteStdin(obj interface{}) error {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	if len(jsonData) >= s.size {
		return os.ErrInvalid
	}
	media.WriteStdinGyroData(jsonData)
	s.data[len(jsonData)] = 0 // null-terminate
	return nil
}

func (s *SharedMemoryWriter) Close() error {
	if err := s.data.Unmap(); err != nil {
		return err
	}
	err := s.file.Close()
	os.Remove(s.file.Name())
	return err
}
