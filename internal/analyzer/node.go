package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// NodeDetector detects Node.js and TypeScript projects.
type NodeDetector struct{}

func (d *NodeDetector) Name() Language { return LangNode }

func (d *NodeDetector) Match(files []string) bool {
	return containsAny(files, "package.json", "tsconfig.json") ||
		hasExtension(files, ".js") || hasExtension(files, ".ts")
}

func (d *NodeDetector) Detect(dir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Language:     LangNode,
		RuntimeImage: RuntimeImage[LangNode],
		Confidence:   0.0,
	}

	// Find JS/TS source files.
	jsFiles, _ := findFiles(dir, []string{"*.js", "*.ts", "*.jsx", "*.tsx", "*.mjs", "*.cjs"})
	if len(jsFiles) == 0 {
		return nil, nil
	}
	result.ProjectFiles = jsFiles
	result.Confidence = 0.5

	// Check for package.json — highest confidence signal.
	pkgPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgPath); err == nil {
		result.DependencyFile = "package.json"
		result.DependencyType = "npm"
		result.InstallCommand = "npm install --production"
		result.Confidence = 0.9

		// Parse package.json for scripts.main and scripts.start.
		data, err := os.ReadFile(pkgPath)
		if err == nil {
			var pkg struct {
				Main    string            `json:"main"`
				Scripts map[string]string `json:"scripts"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				if start, ok := pkg.Scripts["start"]; ok {
					result.Entrypoint = "npm start"
					_ = start // Use npm start which invokes the script.
				} else if pkg.Main != "" {
					result.Entrypoint = "node " + pkg.Main
					result.MainFile = pkg.Main
				}
			}
		}

		// Detect pnpm or yarn.
		if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
			result.DependencyType = "pnpm"
			result.InstallCommand = "pnpm install --prod"
		} else if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
			result.DependencyType = "yarn"
			result.InstallCommand = "yarn install --production"
		}
	}

	// Check for TypeScript config.
	if _, err := os.Stat(filepath.Join(dir, "tsconfig.json")); err == nil {
		result.BuildCommand = "npx tsc"
		result.Confidence += 0.05
	}

	// Detect main entrypoint if not already set.
	if result.Entrypoint == "" {
		mainCandidates := []string{"index.js", "main.js", "app.js", "server.js", "index.ts", "main.ts", "app.ts"}
		for _, candidate := range mainCandidates {
			if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
				result.MainFile = candidate
				if filepath.Ext(candidate) == ".ts" {
					result.Entrypoint = "npx ts-node " + candidate
				} else {
					result.Entrypoint = "node " + candidate
				}
				break
			}
		}
	}

	if result.Entrypoint == "" && len(jsFiles) > 0 {
		result.MainFile = jsFiles[0]
		result.Entrypoint = "node " + jsFiles[0]
	}

	return result, nil
}
