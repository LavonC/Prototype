// Package analyzer provides automatic language detection and Execution Capsule
// packaging for UniCompute. It scans a project directory, identifies the programming
// language, determines the appropriate runtime image, and packages the source code
// into a compressed capsule for remote execution.
package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Language represents a detected programming language.
type Language string

const (
	LangPython  Language = "python"
	LangNode    Language = "node"
	LangCPP     Language = "cpp"
	LangGo      Language = "go"
	LangRust    Language = "rust"
	LangGeneric Language = "generic"
)

// RuntimeImage maps languages to pre-cached runtime container images.
var RuntimeImage = map[Language]string{
	LangPython:  "unicompute/runtime-python:3.11",
	LangNode:    "unicompute/runtime-node:20",
	LangCPP:     "unicompute/runtime-cpp:latest",
	LangGo:      "unicompute/runtime-universal:latest",
	LangRust:    "unicompute/runtime-universal:latest",
	LangGeneric: "unicompute/runtime-universal:latest",
}

// DetectionResult holds the result of scanning a project directory.
type DetectionResult struct {
	// Language is the detected programming language.
	Language Language

	// RuntimeImage is the pre-cached container image to use.
	RuntimeImage string

	// Entrypoint is the suggested entrypoint command.
	Entrypoint string

	// DependencyFile is the path to the dependency manifest (e.g., requirements.txt).
	DependencyFile string

	// DependencyType describes the package manager (e.g., "pip", "npm").
	DependencyType string

	// InstallCommand is the command to install dependencies inside the sandbox.
	InstallCommand string

	// BuildCommand is the command to compile the project (if applicable).
	BuildCommand string

	// MainFile is the detected main/entry source file.
	MainFile string

	// Confidence is how confident the detection is (0.0 to 1.0).
	Confidence float64

	// ProjectFiles lists all relevant source files found.
	ProjectFiles []string
}

// Detector is the interface for language-specific project detection.
type Detector interface {
	// Name returns the language name.
	Name() Language

	// Detect checks if the directory contains this language's project.
	// Returns a DetectionResult and a confidence score.
	Detect(dir string) (*DetectionResult, error)

	// Match checks if any indicator files for this language exist.
	Match(files []string) bool
}

// allDetectors lists all available language detectors in priority order.
var allDetectors = []Detector{
	&PythonDetector{},
	&NodeDetector{},
	&CPPDetector{},
	&GoDetector{},
	&RustDetector{},
	&GenericDetector{},
}

// Analyze scans a directory and returns the best matching DetectionResult.
// It tries all detectors and returns the one with the highest confidence.
func Analyze(dir string) (*DetectionResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absDir)
	}

	// Collect all files in the directory (non-recursive first level for matching).
	topFiles, err := listTopLevelFiles(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var bestResult *DetectionResult
	var bestConfidence float64

	for _, detector := range allDetectors {
		if !detector.Match(topFiles) {
			continue
		}

		result, err := detector.Detect(absDir)
		if err != nil {
			continue
		}

		if result != nil && result.Confidence > bestConfidence {
			bestResult = result
			bestConfidence = result.Confidence
		}
	}

	if bestResult == nil {
		// Fallback to generic.
		generic := &GenericDetector{}
		return generic.Detect(absDir)
	}

	return bestResult, nil
}

// listTopLevelFiles returns the names of all files/dirs in the top level of a directory.
func listTopLevelFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	return files, nil
}

// findFiles recursively finds files matching patterns in a directory.
func findFiles(dir string, patterns []string) ([]string, error) {
	var matches []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access.
		}

		// Skip hidden directories and common non-source directories.
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" ||
				name == "__pycache__" || name == "venv" || name == ".venv" ||
				name == "target" || name == "vendor" || name == "build" ||
				name == "dist" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}

		for _, pattern := range patterns {
			matched, err := filepath.Match(pattern, info.Name())
			if err == nil && matched {
				relPath, _ := filepath.Rel(dir, path)
				matches = append(matches, relPath)
				break
			}
		}

		return nil
	})

	return matches, err
}

// containsAny checks if any of the target strings exist in the list.
func containsAny(list []string, targets ...string) bool {
	for _, item := range list {
		for _, target := range targets {
			if item == target {
				return true
			}
		}
	}
	return false
}
