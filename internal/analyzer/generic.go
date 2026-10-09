package analyzer

// GenericDetector is the fallback detector when no specific language is identified.
type GenericDetector struct{}

func (d *GenericDetector) Name() Language { return LangGeneric }

func (d *GenericDetector) Match(files []string) bool {
	return true // Always matches as a fallback
}

func (d *GenericDetector) Detect(dir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Language:     LangGeneric,
		RuntimeImage: RuntimeImage[LangGeneric],
		Confidence:   0.1,
		Entrypoint:   "/bin/sh -c 'echo \"No specific language detected. Running generic entrypoint.\"; ls -la'",
	}

	// Try to find a shell script to run
	shFiles, _ := findFiles(dir, []string{"*.sh"})
	if len(shFiles) > 0 {
		result.MainFile = shFiles[0]
		for _, f := range shFiles {
			if f == "run.sh" || f == "start.sh" || f == "main.sh" {
				result.MainFile = f
				break
			}
		}
		result.Entrypoint = "sh " + result.MainFile
		result.Confidence = 0.4
	}
	
	result.ProjectFiles = shFiles

	return result, nil
}
