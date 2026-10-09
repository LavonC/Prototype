package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// TransferType identifies the direction/purpose of the file transfer.
type TransferType byte

const (
	TransferTypeCapsule TransferType = 0x10
	TransferTypeArtifact TransferType = 0x20
)

const ChunkSize = 64 * 1024 // 64 KB

// Header format for file transfer chunks:
// Byte 0: TransferType (0x10 or 0x20)
// Byte 1: Flags (0x01 = Start, 0x02 = Data, 0x04 = End)
// Bytes 2-9: Chunk Offset (uint64)
// Bytes 10+: Payload

const (
	FlagStart byte = 0x01
	FlagData  byte = 0x02
	FlagEnd   byte = 0x04
)

// FileTransferManager handles sending and receiving files over a DataChannel.
type FileTransferManager struct {
	dc *DataChannel
	
	mu sync.Mutex
	// Maps transfer type to a receiving file pointer
	receiving map[TransferType]*os.File
}

// NewFileTransferManager creates a new FileTransferManager.
func NewFileTransferManager(dc *DataChannel) *FileTransferManager {
	manager := &FileTransferManager{
		dc:        dc,
		receiving: make(map[TransferType]*os.File),
	}
	
	// Hook into the DataChannel's OnMessage
	dc.SetOnMessage(manager.handleMessage)
	
	return manager
}

// SendFile streams a file over the DataChannel in chunks.
func (m *FileTransferManager) SendFile(filePath string, tType TransferType) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	buffer := make([]byte, ChunkSize)
	offset := uint64(0)
	totalSize := info.Size()

	// Send Start Frame (includes total size as payload)
	startFrame := make([]byte, 18)
	startFrame[0] = byte(tType)
	startFrame[1] = FlagStart
	binary.BigEndian.PutUint64(startFrame[2:10], 0)
	binary.BigEndian.PutUint64(startFrame[10:18], uint64(totalSize))
	
	if err := m.dc.SendBytes(startFrame); err != nil {
		return err
	}

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			frame := make([]byte, 10+n)
			frame[0] = byte(tType)
			frame[1] = FlagData
			binary.BigEndian.PutUint64(frame[2:10], offset)
			copy(frame[10:], buffer[:n])
			
			if sendErr := m.dc.SendBytes(frame); sendErr != nil {
				return sendErr
			}
			
			offset += uint64(n)
			
			// Simple backpressure logic: Wait if buffered amount is high
			if m.dc.dc.BufferedAmount() > 4*1024*1024 { // 4MB
				time.Sleep(10 * time.Millisecond)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Send End Frame
	endFrame := make([]byte, 10)
	endFrame[0] = byte(tType)
	endFrame[1] = FlagEnd
	binary.BigEndian.PutUint64(endFrame[2:10], offset)
	
	return m.dc.SendBytes(endFrame)
}

func (m *FileTransferManager) handleMessage(data []byte) {
	if len(data) < 10 {
		return // Invalid frame
	}

	tType := TransferType(data[0])
	flags := data[1]
	offset := binary.BigEndian.Uint64(data[2:10])

	m.mu.Lock()
	defer m.mu.Unlock()

	switch flags {
	case FlagStart:
		// totalSize := binary.BigEndian.Uint64(data[10:18])
		// Initialize temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("unicompute_transfer_%d_*", tType))
		if err == nil {
			m.receiving[tType] = tmpFile
		}
	case FlagData:
		file, ok := m.receiving[tType]
		if ok {
			file.WriteAt(data[10:], int64(offset))
		}
	case FlagEnd:
		file, ok := m.receiving[tType]
		if ok {
			file.Close()
			// Notify completion (In a real implementation, trigger a callback or channel)
			// e.g., OnTransferComplete(tType, file.Name())
			delete(m.receiving, tType)
		}
	}
}
