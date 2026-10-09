package requester

import (
	"os"
	"path/filepath"
	"sync"
	
	"github.com/rs/zerolog/log"
)

// Aggregator manages the collection and merging of returned artifacts.
type Aggregator struct {
	outputDir string
	
	mu              sync.Mutex
	sourceCapsules  map[string]string // TaskID -> Local Path
	mergedArtifacts map[string]string // JobID -> Merged Path
}

// NewAggregator creates a new Aggregator.
func NewAggregator(outDir string) *Aggregator {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatal().Err(err).Msg("Failed to create aggregator output directory")
	}
	
	return &Aggregator{
		outputDir:       outDir,
		sourceCapsules:  make(map[string]string),
		mergedArtifacts: make(map[string]string),
	}
}

// RegisterSourceCapsule stores the path to the original capsule so we can stream it to workers.
func (a *Aggregator) RegisterSourceCapsule(taskID, path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sourceCapsules[taskID] = path
}

// GetSourceCapsule returns the path to the capsule for a task.
func (a *Aggregator) GetSourceCapsule(taskID string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	path, ok := a.sourceCapsules[taskID]
	return path, ok
}

// MergeJobArtifacts combines multiple chunked results into a single output file (e.g. for map/reduce).
func (a *Aggregator) MergeJobArtifacts(jobID string, artifacts []string) (string, error) {
	if len(artifacts) == 0 {
		return "", nil // No artifacts to merge
	}
	
	// This is a naive merge. A real implementation would parse the artifacts 
	// (JSON arrays, CSV rows, etc.) and concatenate them properly.
	
	mergedPath := filepath.Join(a.outputDir, jobID+"_merged_results.dat")
	out, err := os.Create(mergedPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	
	for _, artPath := range artifacts {
		in, err := os.Open(artPath)
		if err != nil {
			log.Warn().Err(err).Str("artifact", artPath).Msg("Failed to open artifact for merging, skipping")
			continue
		}
		
		_, err = out.ReadFrom(in)
		in.Close()
		if err != nil {
			return "", err
		}
	}
	
	a.mu.Lock()
	a.mergedArtifacts[jobID] = mergedPath
	a.mu.Unlock()
	
	return mergedPath, nil
}
