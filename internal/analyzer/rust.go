package analyzer

import (
	"os"
	"path/filepath"
)

// RustDetector detects Rust projects.
type RustDetector struct{}

func (d *RustDetector) Name() Language { return LangRust }

func (d *RustDetector) Match(files []string) bool {
	return containsAny(files, "Cargo.toml") || hasExtension(files, ".rs")
}

func (d *RustDetector) Detect(dir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Language:     LangRust,
		RuntimeImage: RuntimeImage[LangRust],
		Confidence:   0.0,
	}

	rsFiles, _ := findFiles(dir, []string{"*.rs"})
	if len(rsFiles) == 0 {
		return nil, nil
	}
	result.ProjectFiles = rsFiles
	result.Confidence = 0.5

	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		result.DependencyFile = "Cargo.toml"
		result.DependencyType = "cargo"
		result.BuildCommand = "cargo build --release"
		result.Entrypoint = "./target/release/app" // Assuming package name is 'app' or changed by packager
		result.Confidence = 0.9
	} else {
		// Fallback to rustc
		if len(rsFiles) > 0 {
			result.MainFile = rsFiles[0]
			for _, f := range rsFiles {
				if filepath.Base(f) == "main.rs" {
					result.MainFile = f
					break
				}
			}
			result.BuildCommand = "rustc -O " + result.MainFile + " -o app"
			result.Entrypoint = "./app"
		}
	}

	return result, nil
}
