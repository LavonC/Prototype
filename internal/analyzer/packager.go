package analyzer

import (
	"fmt"
	"path/filepath"

	"github.com/unicompute/unicompute/internal/capsule"
	"github.com/unicompute/unicompute/internal/protocol"
)

// Packager coordinates the detection and compression of a project.
type Packager struct {
	MaxSizeBytes int64
}

// NewPackager creates a new Packager with a size limit.
func NewPackager(maxSizeMB int) *Packager {
	return &Packager{
		MaxSizeBytes: int64(maxSizeMB) * 1024 * 1024,
	}
}

// PackageProject scans a directory, identifies the language, compresses the source,
// and generates a complete CapsuleManifest.
func (p *Packager) PackageProject(sourceDir, outputDir string) (*protocol.CapsuleManifest, string, error) {
	// 1. Detect language and requirements
	result, err := Analyze(sourceDir)
	if err != nil {
		return nil, "", fmt.Errorf("analysis failed: %w", err)
	}

	// 2. Compress the project files into a tar.gz archive
	archivePath := filepath.Join(outputDir, "payload.tar.gz")
	
	files, totalSize, err := capsule.Compress(sourceDir, archivePath)
	if err != nil {
		return nil, "", fmt.Errorf("compression failed: %w", err)
	}

	if totalSize > p.MaxSizeBytes {
		return nil, "", fmt.Errorf("project size %d exceeds maximum allowed %d", totalSize, p.MaxSizeBytes)
	}

	// 3. Prepare dependency info
	var depInfo *protocol.DependencyInfo
	if result.DependencyFile != "" {
		depInfo = &protocol.DependencyInfo{
			Type:    result.DependencyType,
			File:    result.DependencyFile,
			Command: result.InstallCommand,
		}
	}

	// 4. If there's a build step needed before the entrypoint,
	// we combine it into a single shell command for the entrypoint.
	finalEntrypoint := result.Entrypoint
	if result.BuildCommand != "" {
		finalEntrypoint = result.BuildCommand + " && " + result.Entrypoint
	}

	// 5. Generate Manifest
	manifest, err := capsule.GenerateManifest(
		files,
		totalSize,
		string(result.Language),
		result.RuntimeImage,
		finalEntrypoint,
		depInfo,
		archivePath,
	)
	if err != nil {
		return nil, "", fmt.Errorf("manifest generation failed: %w", err)
	}

	manifestPath := filepath.Join(outputDir, "manifest.json")
	if err := capsule.WriteManifest(manifest, manifestPath); err != nil {
		return nil, "", fmt.Errorf("failed to write manifest: %w", err)
	}

	return manifest, archivePath, nil
}
