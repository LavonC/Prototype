package capsule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/unicompute/unicompute/internal/protocol"
)

// GenerateManifest creates a complete manifest for an Execution Capsule.
func GenerateManifest(files []protocol.FileEntry, totalSize int64, lang string, runtime string, entrypoint string, depInfo *protocol.DependencyInfo, archiveFile string) (*protocol.CapsuleManifest, error) {
	// Calculate the SHA-256 hash of the final archive file
	archiveHash, err := hashFile(archiveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to hash archive: %w", err)
	}

	manifest := &protocol.CapsuleManifest{
		CapsuleID:        "cap-" + uuid.New().String()[:8],
		DetectedLanguage: lang,
		RuntimeImage:     runtime,
		Entrypoint:       entrypoint,
		TotalSizeBytes:   totalSize,
		FileCount:        len(files),
		Files:            files,
		Dependencies:     depInfo,
		SHA256:           archiveHash,
	}

	return manifest, nil
}

// WriteManifest saves the manifest to a JSON file.
func WriteManifest(manifest *protocol.CapsuleManifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadManifest loads a manifest from a JSON file.
func ReadManifest(path string) (*protocol.CapsuleManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest protocol.CapsuleManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
