package analyzer

import (
	"os"
	"path/filepath"
)

// CPPDetector detects C and C++ projects.
type CPPDetector struct{}

func (d *CPPDetector) Name() Language { return LangCPP }

func (d *CPPDetector) Match(files []string) bool {
	return containsAny(files, "Makefile", "CMakeLists.txt", "meson.build") ||
		hasExtension(files, ".c") || hasExtension(files, ".cpp") ||
		hasExtension(files, ".cc") || hasExtension(files, ".h") ||
		hasExtension(files, ".hpp")
}

func (d *CPPDetector) Detect(dir string) (*DetectionResult, error) {
	result := &DetectionResult{
		Language:     LangCPP,
		RuntimeImage: RuntimeImage[LangCPP],
		Confidence:   0.0,
	}

	// Find C/C++ source files.
	srcFiles, _ := findFiles(dir, []string{"*.c", "*.cpp", "*.cc", "*.cxx"})
	headerFiles, _ := findFiles(dir, []string{"*.h", "*.hpp", "*.hxx"})
	allFiles := append(srcFiles, headerFiles...)

	if len(srcFiles) == 0 {
		return nil, nil
	}
	result.ProjectFiles = allFiles
	result.Confidence = 0.5

	// Detect build system.
	if _, err := os.Stat(filepath.Join(dir, "CMakeLists.txt")); err == nil {
		result.DependencyType = "cmake"
		result.BuildCommand = "mkdir -p build && cd build && cmake .. -DCMAKE_BUILD_TYPE=Release && make -j$(nproc)"
		result.Entrypoint = "./build/app"
		result.Confidence = 0.9
	} else if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
		result.DependencyType = "make"
		result.BuildCommand = "make -j$(nproc)"
		result.Entrypoint = "./app"
		result.Confidence = 0.85
	} else if _, err := os.Stat(filepath.Join(dir, "meson.build")); err == nil {
		result.DependencyType = "meson"
		result.BuildCommand = "meson setup build && cd build && ninja"
		result.Entrypoint = "./build/app"
		result.Confidence = 0.85
	} else {
		// No build system — compile directly.
		if hasCPPFiles(srcFiles) {
			result.BuildCommand = "g++ -O3 -std=c++17 -o app " + joinSourceFiles(srcFiles, ".cpp", ".cc", ".cxx")
		} else {
			result.BuildCommand = "gcc -O3 -o app " + joinSourceFiles(srcFiles, ".c")
		}
		result.Entrypoint = "./app"
		result.Confidence = 0.6
	}

	// Detect main file.
	mainCandidates := []string{"main.cpp", "main.c", "app.cpp", "app.c"}
	for _, candidate := range mainCandidates {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			result.MainFile = candidate
			break
		}
	}

	return result, nil
}

// hasCPPFiles checks if any C++ source files exist in the list.
func hasCPPFiles(files []string) bool {
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
			return true
		}
	}
	return false
}

// joinSourceFiles joins source files with matching extensions into a space-separated string.
func joinSourceFiles(files []string, extensions ...string) string {
	result := ""
	for _, f := range files {
		ext := filepath.Ext(f)
		for _, e := range extensions {
			if ext == e {
				if result != "" {
					result += " "
				}
				result += f
				break
			}
		}
	}
	if result == "" {
		// Fallback: include all source files.
		for _, f := range files {
			if result != "" {
				result += " "
			}
			result += f
		}
	}
	return result
}
