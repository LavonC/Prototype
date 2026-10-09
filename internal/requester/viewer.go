package requester

import (
	"fmt"
	"os"
	"sync"
)

// Viewer multiplexes and displays live stdout/stderr streams from multiple workers.
type Viewer struct {
	mu sync.Mutex
	// In a real terminal UI (like tview or bubbletea), this would map to UI panes.
	// For this abstraction, we just prefix lines with the TaskID.
}

// NewViewer initializes the live stream viewer.
func NewViewer() *Viewer {
	return &Viewer{}
}

// HandleStreamData processes incoming stream chunks from Channel A.
func (v *Viewer) HandleStreamData(taskID string, isError bool, data []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Split by newline and prefix
	// (Simplified for simulation: just print raw with prefix)
	
	prefix := fmt.Sprintf("[%s]", taskID)
	if isError {
		fmt.Fprintf(os.Stderr, "\033[31m%s \033[0m%s", prefix, string(data))
	} else {
		fmt.Fprintf(os.Stdout, "\033[36m%s \033[0m%s", prefix, string(data))
	}
}
