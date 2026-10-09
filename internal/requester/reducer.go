package requester

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// Reducer handles executing the final aggregation step locally on the Requester's machine.
type Reducer struct {
	outputDir string
}

// NewReducer creates a new reducer.
func NewReducer(outputDir string) *Reducer {
	return &Reducer{
		outputDir: outputDir,
	}
}

// ExecuteReduce takes the path to the combined output file and runs the reduce script on it.
// e.g. reduceCommand = "python reduce.py"
func (r *Reducer) ExecuteReduce(ctx context.Context, reduceCommand string, mergedDataPath string) (string, error) {
	log.Info().Str("merged_data", mergedDataPath).Msg("Starting local reduce phase")
	
	finalOutputPath := filepath.Join(r.outputDir, "final_result.out")
	
	// Create the full shell command. We pass the merged data as an argument.
	// For cross-platform, we use sh/cmd abstractions. 
	// Assuming UNIX-like for this backend logic execution.
	cmdStr := fmt.Sprintf("%s %s > %s", reduceCommand, mergedDataPath, finalOutputPath)
	
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(output)).Msg("Reduce script failed")
		return "", fmt.Errorf("reduce failed: %w (output: %s)", err, string(output))
	}
	
	log.Info().Str("final_output", finalOutputPath).Msg("Reduce phase completed successfully")
	
	// Cleanup the intermediate merged file to save space
	os.Remove(mergedDataPath)
	
	return finalOutputPath, nil
}
