package api

import (
	"fmt"
	"strings"
	"time"
)

// ValidateTaskSpec validates a TaskSpec and returns any errors.
func ValidateTaskSpec(spec *TaskSpec) error {
	var errs []string

	if strings.TrimSpace(spec.Entrypoint) == "" {
		errs = append(errs, "entrypoint is required")
	}

	if spec.RequiredCPUs < 0 {
		errs = append(errs, "required_cpus must be non-negative")
	}
	if spec.RequiredCPUs > 128 {
		errs = append(errs, "required_cpus exceeds maximum of 128")
	}

	if spec.RequiredMemoryMB < 0 {
		errs = append(errs, "required_memory_mb must be non-negative")
	}
	if spec.RequiredMemoryMB > 1024*1024 {
		errs = append(errs, "required_memory_mb exceeds maximum of 1TB")
	}

	if spec.Timeout < 0 {
		errs = append(errs, "timeout must be non-negative")
	}
	if spec.Timeout > 24*time.Hour {
		errs = append(errs, "timeout exceeds maximum of 24 hours")
	}

	if spec.Priority < 0 || spec.Priority > 100 {
		errs = append(errs, "priority must be between 0 and 100")
	}

	if len(errs) > 0 {
		return fmt.Errorf("task spec validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateMapSpec validates a MapSpec and returns any errors.
func ValidateMapSpec(spec *MapSpec) error {
	var errs []string

	if strings.TrimSpace(spec.Entrypoint) == "" {
		errs = append(errs, "entrypoint is required")
	}

	if !strings.Contains(spec.Entrypoint, "{{item}}") {
		errs = append(errs, "entrypoint must contain {{item}} placeholder")
	}

	if len(spec.Items) == 0 {
		errs = append(errs, "at least one item is required")
	}

	if spec.Concurrency < 0 {
		errs = append(errs, "concurrency must be non-negative")
	}
	if spec.Concurrency > 1000 {
		errs = append(errs, "concurrency exceeds maximum of 1000")
	}

	if err := ValidateTaskSpec(&spec.TaskSpec); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("map spec validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
