package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// VersionInfo holds the released and current versions for an artifact.
type VersionInfo struct {
	ReleasedVersion string
	CurrentVersion  string
}

// LoadVersionsFromFile reads a file with lines in the format "artifact-id:released-version:current-version"
// and returns a map with artifact-id as the key and VersionInfo as the value.
// It skips empty lines and lines starting with '#'.
func LoadVersionsFromFile(filePath string) (map[string]VersionInfo, error) {
	versionsMap := make(map[string]VersionInfo)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines or comment lines (starting with #)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		parts := strings.Split(trimmedLine, ":")
		if len(parts) != 3 {
			fmt.Printf("Warning: Skipping malformed line %d in %s: Expected 3 parts, got %d. Line: '%s'\n", lineNumber, filePath, len(parts), trimmedLine)
			continue
		}

		artifactId := strings.TrimSpace(parts[0])
		releasedVersion := strings.TrimSpace(parts[1])
		currentVersion := strings.TrimSpace(parts[2])

		if artifactId == "" {
			fmt.Printf("Warning: Skipping line %d in %s due to empty artifact-id. Line: '%s'\n", lineNumber, filePath, trimmedLine)
			continue
		}

		versionsMap[artifactId] = VersionInfo{
			ReleasedVersion: releasedVersion,
			CurrentVersion:  currentVersion,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %w", filePath, err)
	}

	return versionsMap, nil
}
