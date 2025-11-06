package checker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/drape-io/chex/internal/config"
)

// Result represents the result of checking a single tool.
type Result struct {
	Tool             *config.Tool
	Status           Status
	InstalledVersion string
	Path             string
	Output           string
	Error            error
}

// Status represents the check status.
type Status string

const (
	StatusPass            Status = "pass"
	StatusFail            Status = "fail"
	StatusOptionalMissing Status = "optional_missing"
)

// Check checks a single tool and returns the result.
func Check(tool *config.Tool) *Result {
	result := &Result{
		Tool: tool,
	}

	// If no version specified, just check existence
	if tool.Version == "" {
		return checkExistence(tool, result)
	}

	// Version specified, check version
	return checkVersion(tool, result)
}

// checkExistence checks if a tool exists on PATH without executing it.
func checkExistence(tool *config.Tool, result *Result) *Result {
	path, err := exec.LookPath(tool.CLI)
	if err != nil {
		result.Status = StatusFail
		if tool.Optional {
			result.Status = StatusOptionalMissing
		}
		result.Error = fmt.Errorf("%s: command not found", tool.CLI)
		return result
	}

	result.Status = StatusPass
	result.Path = path
	return result
}

// checkVersion checks if a tool exists and matches the version constraint.
func checkVersion(tool *config.Tool, result *Result) *Result {
	// Execute command to get version
	versionOutput, err := executeVersionCommand(tool)
	if err != nil {
		result.Status = StatusFail
		if tool.Optional {
			result.Status = StatusOptionalMissing
		}
		result.Error = err
		return result
	}

	result.Output = versionOutput

	// Extract version from output
	version, err := extractVersion(versionOutput, tool.VersionPattern)
	if err != nil {
		result.Status = StatusFail
		result.Error = fmt.Errorf("failed to extract version: %w", err)
		return result
	}

	result.InstalledVersion = version

	// Parse and check version constraint
	constraint, err := semver.NewConstraint(tool.Version)
	if err != nil {
		result.Status = StatusFail
		result.Error = fmt.Errorf("invalid version constraint '%s': %w", tool.Version, err)
		return result
	}

	installedVer, err := semver.NewVersion(version)
	if err != nil {
		result.Status = StatusFail
		result.Error = fmt.Errorf("failed to parse installed version '%s': %w", version, err)
		return result
	}

	if constraint.Check(installedVer) {
		result.Status = StatusPass
	} else {
		result.Status = StatusFail
		if tool.Optional {
			result.Status = StatusOptionalMissing
		}
	}

	return result
}

// executeVersionCommand executes the tool with its version argument.
func executeVersionCommand(tool *config.Tool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If version arg is specified, use it
	if tool.VersionArg != "" {
		args := strings.Fields(tool.VersionArg)
		cmd := exec.CommandContext(ctx, tool.CLI, args...)
		output, err := runCommand(cmd)
		if err != nil {
			return "", fmt.Errorf("%s: %w", tool.CLI, err)
		}
		return output, nil
	}

	// Smart guessing: try common version arguments
	// Order matters: try most common first
	commonVersionArgs := [][]string{
		{"--version"},
		{"version"},
		{"-v"},
		{"-V"},
		{"--version-short"},
		{"-version"},
	}

	for _, args := range commonVersionArgs {
		cmd := exec.CommandContext(ctx, tool.CLI, args...)
		output, err := runCommand(cmd)

		// If we got output with version-like content, use it (even if exit code was non-zero)
		// Some tools (like kubeconform -v) may exit with non-zero but still print version
		if output != "" && looksLikeVersionOutput(output) {
			return output, nil
		}

		// Continue trying other args regardless of error
		_ = err // Ignore error, try next arg
	}

	return "", fmt.Errorf("%s: failed to get version", tool.CLI)
}

// looksLikeVersionOutput checks if output looks like version information.
func looksLikeVersionOutput(output string) bool {
	// Check if output contains version-like patterns
	// Should have digits and dots, and be reasonably short
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return false
	}

	firstLine := strings.TrimSpace(lines[0])
	lowerFirstLine := strings.ToLower(firstLine)

	// Should be relatively short (version output is usually concise)
	if len(firstLine) > 200 {
		return false
	}

	// Reject obvious help/error messages
	rejectPatterns := []string{
		"usage:",
		"error:",
		"flag provided",
		"unknown flag",
		"unknown command",
		"help",
		"failed validation",
	}
	for _, pattern := range rejectPatterns {
		if strings.Contains(lowerFirstLine, pattern) {
			return false
		}
	}

	// Should contain at least one digit
	hasDigit := false
	for _, char := range firstLine {
		if char >= '0' && char <= '9' {
			hasDigit = true
			break
		}
	}

	// Version output usually contains version number patterns (X.Y or vX.Y)
	// Look for common version patterns
	hasVersionPattern := strings.Contains(firstLine, ".") && hasDigit

	return hasVersionPattern
}

// runCommand runs a command and returns combined stdout/stderr.
func runCommand(cmd *exec.Cmd) (string, error) {
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	if err != nil {
		if output != "" {
			return output, err
		}
		return "", err
	}

	return output, nil
}

// extractVersion extracts a version string from command output.
func extractVersion(output, pattern string) (string, error) {
	if pattern != "" {
		// Use custom pattern
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid version pattern: %w", err)
		}
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			return matches[1], nil
		}
		if len(matches) > 0 {
			return matches[0], nil
		}
		return "", errors.New("pattern did not match")
	}

	// Default: find first line with version-like pattern (X.Y.Z or X.Y)
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to find version pattern: X.Y.Z or X.Y
		versionRe := regexp.MustCompile(`\d+\.\d+(\.\d+)?`)
		match := versionRe.FindString(line)
		if match != "" {
			return match, nil
		}
	}

	return "", errors.New("no version found in output")
}

// CheckAll checks multiple tools and returns their results.
func CheckAll(tools map[string]*config.Tool, filter []string) []*Result {
	var results []*Result

	// If filter is provided, only check those tools
	if len(filter) > 0 {
		for _, name := range filter {
			tool, exists := tools[name]
			if !exists {
				// Tool not found in config
				results = append(results, &Result{
					Tool: &config.Tool{
						Name: name,
						CLI:  name,
					},
					Status: StatusFail,
					Error:  fmt.Errorf("tool '%s' not found in configuration", name),
				})
				continue
			}
			results = append(results, Check(tool))
		}
	} else {
		// Check all tools
		for _, tool := range tools {
			results = append(results, Check(tool))
		}
	}

	return results
}
