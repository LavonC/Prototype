package requester

import (
	"fmt"
	"math"
)

// MapSplitter breaks a large dataset or iteration space into smaller chunks
// that can be executed in parallel by different workers.
type MapSplitter struct {
	concurrency int
}

// NewMapSplitter creates a splitter tailored to the requested concurrency.
func NewMapSplitter(concurrency int) *MapSplitter {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &MapSplitter{concurrency: concurrency}
}

// SplitArray calculates the start and end indices for partitioning an array
// of size N into M concurrent chunks.
func (s *MapSplitter) SplitArray(totalItems int) [][2]int {
	if totalItems == 0 {
		return nil
	}

	chunks := s.concurrency
	if totalItems < chunks {
		chunks = totalItems
	}

	chunkSize := float64(totalItems) / float64(chunks)
	var partitions [][2]int

	currentStart := 0
	for i := 0; i < chunks; i++ {
		// Calculate the end index (exclusive) for this chunk
		currentEnd := int(math.Round(float64(i+1) * chunkSize))
		if currentEnd > totalItems {
			currentEnd = totalItems
		}
		
		partitions = append(partitions, [2]int{currentStart, currentEnd})
		currentStart = currentEnd
	}

	return partitions
}

// GenerateCommands creates a list of shell commands based on the partitions.
// E.g. baseCommand="python process.py", partitions=[[0, 10], [10, 20]]
// Output: ["python process.py 0 10", "python process.py 10 20"]
func (s *MapSplitter) GenerateCommands(baseCommand string, partitions [][2]int) []string {
	var cmds []string
	for _, p := range partitions {
		cmds = append(cmds, fmt.Sprintf("%s %d %d", baseCommand, p[0], p[1]))
	}
	return cmds
}
