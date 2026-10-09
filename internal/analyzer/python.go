package analyzer

import (
	"os"
	"path/filepath"
)

// PythonDetector detects Python projects.
type PythonDetector struct{}

func (d *PythonDetector) Name() Language { return LangPython }

func (d *PythonDetector) Match(files []string) bool {
	return containsAny(files,
		"requirements.txt", "setup.py", "setup.cfg", "pyproject.toml",
		"Pipfile", "environment.yml", "conda.yml",
	) || hasExtension(files, ".py")
}

func (d *PythonDetector) Detect(dir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Language:     LangPython,
		RuntimeImage: RuntimeImage[LangPython],
		Confidence:   0.0,
	}

	// Find Python source files.
	pyFiles, err := findFiles(dir, []string{"*.py"})
	if err != nil {
		return nil, err
	}
	if len(pyFiles) == 0 {
		return nil, nil
	}
	result.ProjectFiles = pyFiles
	result.Confidence = 0.5

	// Detect dependency file.
	depFiles := map[string]struct {
		depType    string
		installCmd string
		confidence float64
	}{
		"requirements.txt": {"pip", "pip install --no-cache-dir -r requirements.txt", 0.9},
		"pyproject.toml":   {"pip", "pip install --no-cache-dir .", 0.85},
		"setup.py":         {"pip", "pip install --no-cache-dir .", 0.8},
		"Pipfile":          {"pipenv", "pipenv install --deploy", 0.85},
		"environment.yml":  {"conda", "conda env update -f environment.yml", 0.8},
	}

	for file, info := range depFiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			result.DependencyFile = file
			result.DependencyType = info.depType
			result.InstallCommand = info.installCmd
			result.Confidence = info.confidence
			break
		}
	}

	// Detect main entrypoint.
	mainCandidates := []string{"main.py", "app.py", "run.py", "__main__.py", "train.py", "inference.py"}
	for _, candidate := range mainCandidates {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			result.MainFile = candidate
			result.Entrypoint = "python3 " + candidate
			result.Confidence += 0.1
			break
		}
	}

	// If no main file found, use the first .py file.
	if result.MainFile == "" && len(pyFiles) > 0 {
		result.MainFile = pyFiles[0]
		result.Entrypoint = "python3 " + pyFiles[0]
	}

	return result, nil
}

func hasExtension(files []string, ext string) bool {
	for _, f := range files {
		if filepath.Ext(f) == ext {
			return true
		}
	}
	return false
}
