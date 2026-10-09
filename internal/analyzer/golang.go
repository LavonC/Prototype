package analyzer

import (
	"os"
	"path/filepath"
)

// GoDetector detects Go projects.
type GoDetector struct{}

func (d *GoDetector) Name() Language { return LangGo }

func (d *GoDetector) Match(files []string) bool {
	return containsAny(files, "go.mod") || hasExtension(files, ".go")
}

func (d *GoDetector) Detect(dir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Language:     LangGo,
		RuntimeImage: RuntimeImage[LangGo],
		Confidence:   0.0,
	}

	goFiles, _ := findFiles(dir, []string{"*.go"})
	if len(goFiles) == 0 {
		return nil, nil
	}
	result.ProjectFiles = goFiles
	result.Confidence = 0.5

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		result.DependencyFile = "go.mod"
		result.DependencyType = "go-mod"
		result.InstallCommand = "go mod download"
		result.Confidence = 0.9
	}

	result.BuildCommand = "go build -o app"
	result.Entrypoint = "./app"

	mainCandidates := []string{"main.go", "cmd/main.go"}
	for _, candidate := range mainCandidates {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			result.MainFile = candidate
			result.BuildCommand = "go build -o app " + candidate
			break
		}
	}

	return result, nil
}
