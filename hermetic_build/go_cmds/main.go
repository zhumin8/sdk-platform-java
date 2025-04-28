package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// OwlBotHermeticConfig represents the structure of the .OwlBot-hermetic.yaml file.
type OwlBotHermeticConfig struct {
	DeepRemoveRegex   []string `yaml:"deep-remove-regex"`
	DeepPreserveRegex []string `yaml:"deep-preserve-regex"`
}

// CleanCommand defines the structure for the clean command.
type CleanCommand struct {
	RepoRoot        string
	LibraryID       string
	DryRun          bool
	additionalFiles string
	flagSet         *flag.FlagSet // To hold the flags for this command
}

// NewCleanCommand creates a new CleanCommand instance.
func NewCleanCommand() *CleanCommand {
	cmd := &CleanCommand{
		flagSet: flag.NewFlagSet("clean", flag.ExitOnError), // Initialize the flag set
	}
	cmd.initFlags() // Define flags
	return cmd
}

// initFlags defines the flags for the clean command.
func (c *CleanCommand) initFlags() {
	c.flagSet.StringVar(&c.RepoRoot, "repo-root", "", "Path to the root of the clone")
	c.flagSet.StringVar(&c.LibraryID, "library-id", "", "ID of the library to clean generated files from, e.g., java-accesscontextmanager")
	c.flagSet.BoolVar(&c.DryRun, "dry-run", false, "List files to remove without actually deleting them")
	c.flagSet.StringVar(&c.additionalFiles, "library-files", ".repo-metadata.json", "Comma-separated list of files to remove from library root")
	// Set custom usage function for this flag set.
	c.flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s [options]\n", os.Args[0], c.flagSet.Name())
		fmt.Fprintln(os.Stderr, "Options:")
		c.flagSet.PrintDefaults()
	}
}

// Execute runs the clean command.
func (c *CleanCommand) Execute(args []string) error {
	// Parse the flags.
	if err := c.flagSet.Parse(args); err != nil {
		return err
	}

	// Validate that the required flags are set.
	if c.RepoRoot == "" || c.LibraryID == "" {
		fmt.Println("Error: Both -repo-root and -library-id are required.")
		c.flagSet.Usage()
		return fmt.Errorf("required flags not set") // Use a proper error
	}

	fmt.Printf("Cleaning library %s in repository root: %s\n", c.LibraryID, c.RepoRoot)

	//  start cleaning logic
	libraryPath := filepath.Join(c.RepoRoot, c.LibraryID)

	config, err := c.loadOwlBotConfig(libraryPath)
	if err != nil {
		return err
	}

	fmt.Printf("Repository Root: %s\n", c.RepoRoot)
	fmt.Printf("Library ID: %s\n", c.LibraryID)
	fmt.Printf("Dry-run mode: %v\n", c.DryRun)
	fmt.Printf("Additional files to remove: %s\n", c.additionalFiles)

	removedSpecificFiles := c.removeSpecificFilesFromRoot(libraryPath)

	if len(config.DeepRemoveRegex) > 0 {
		fmt.Println("Processing deep-remove-regex for files...")
		err := c.performRegexRemoval(libraryPath, c.LibraryID, config.DeepRemoveRegex, config.DeepPreserveRegex)
		if err != nil {
			return err
		}
		fmt.Println("File removal complete. Cleaning up empty directories...")
		_, err = c.cleanupEmptyDirs(libraryPath)
		if err != nil {
			return fmt.Errorf("error during empty directory cleanup: %w", err)
		}
		fmt.Println("Empty directory cleanup complete.")
	} else {
		fmt.Println("No deep-remove-regex found in .OwlBot-hermetic.yaml. Nothing to remove.")
	}

	fmt.Printf("Total cleanup completed. Removed %d specific files from root.\n", removedSpecificFiles)
	return nil
}

// loadOwlBotConfig reads and parses the .OwlBot-hermetic.yaml file.
func (c *CleanCommand) loadOwlBotConfig(libraryPath string) (*OwlBotHermeticConfig, error) {
	configPath := filepath.Join(libraryPath, ".OwlBot-hermetic.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Warning: %s not found in %s. No deep removal will be performed.\n", ".OwlBot-hermetic.yaml", libraryPath)
			return &OwlBotHermeticConfig{}, nil // Return an empty config
		}
		return nil, fmt.Errorf("error reading %s: %w", configPath, err)
	}

	var config OwlBotHermeticConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling %s: %w", configPath, err)
	}

	return &config, nil
}

func (c *CleanCommand) getRelativePattern(pattern string, libraryID string) string {
	relativePattern := pattern
	if len(pattern) > 0 && pattern[0] == '/' {
		if len(pattern) > len(fmt.Sprintf("/%s/", libraryID)) && pattern[:len(fmt.Sprintf("/%s/", libraryID))] == fmt.Sprintf("/%s/", libraryID) {
			relativePattern = pattern[len(fmt.Sprintf("/%s/", libraryID)):]
		} else if len(pattern) > len(fmt.Sprintf("/%s", libraryID)) && pattern[:len(fmt.Sprintf("/%s", libraryID))] == fmt.Sprintf("/%s", libraryID) {
			relativePattern = pattern[len(fmt.Sprintf("/%s", libraryID)):]
		} else {
			relativePattern = pattern[1:]
		}
	}
	return relativePattern
}

// shouldPreserve checks if the given path matches any of the preserve regexes.
func (c *CleanCommand) shouldPreserve(relativePath string, libraryID string, preserveRegexes []string) bool {
	for _, pattern := range preserveRegexes {
		relativePattern := c.getRelativePattern(pattern, libraryID)
		re, err := regexp.Compile(relativePattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Invalid preserve regex '%s': %v\n", pattern, err)
			continue
		}
		if re.MatchString(relativePath) {
			return true
		}
	}
	return false
}

// performRegexRemoval reads deep-remove-regex and removes matching paths within the libraryPath.
func (c *CleanCommand) performRegexRemoval(libraryPath string, libraryID string, removeRegexes []string, preserveRegexes []string) error {
	matchesToRemove := []string{}

	for _, pattern := range removeRegexes {
		relativePattern := c.getRelativePattern(pattern, libraryID)

		re, err := regexp.Compile(relativePattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Invalid regex '%s' (relative '%s'): %v\n", pattern, relativePattern, err)
			continue
		}

		err = filepath.Walk(libraryPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == libraryPath || info.IsDir() {
				return nil // Skip the libraryPath itself and directories for now
			}

			relativePath := filepath.ToSlash(filepath.Clean(path[len(libraryPath)+1:]))

			// if relativePath == "google-identity-accesscontextmanager/src/test/java/com/google/identity/accesscontextmanager/v1/it/ITFakeTest.java" {
			// 	fmt.Printf("break here to debug")
			// }
			if re.MatchString(relativePath) {

				if !c.shouldPreserve(relativePath, libraryID, preserveRegexes) {
					matchesToRemove = append(matchesToRemove, path)
				} else {
					fmt.Printf("[Preserved]: %s (matches preserve regex)\n", path)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error during regex matching for pattern '%s': %w", pattern, err)
		}
	}

	// Sort the matches in reverse order (deepest first)
	sort.Slice(matchesToRemove, func(i, j int) bool {
		return len(strings.Split(matchesToRemove[i], string(filepath.Separator))) > len(strings.Split(matchesToRemove[j], string(filepath.Separator)))
	})

	removedCount := 0
	// Remove the matched paths
	for _, pathToRemove := range matchesToRemove {
		action := "Would remove"
		if !c.DryRun {
			action = "Removing"
			err := os.RemoveAll(pathToRemove)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error removing '%s': %v\n", pathToRemove, err)
			} else {
				removedCount++
			}
		}
		fmt.Printf("[%s]: %s\n", action, pathToRemove)
	}
	fmt.Printf("Deep removal process complete. Total files/directories processed for removal: %d\n", len(matchesToRemove))
	if !c.DryRun {
		fmt.Printf("Total files/directories actually removed: %d\n", removedCount)
	}
	return nil
}

// cleanupEmptyDirs recursively removes empty directories.
func (c *CleanCommand) cleanupEmptyDirs(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		if !c.DryRun {
			fmt.Printf("[Removing Empty Dir]: %s\n", dir)
			err := os.Remove(dir)
			return err == nil, err
		}
		fmt.Printf("[Would Remove Empty Dir]: %s\n", dir)
		return true, nil
	}

	// Recursively cleanup subdirectories
	allEmpty := true
	for _, entry := range entries {
		if entry.IsDir() {
			empty, err := c.cleanupEmptyDirs(filepath.Join(dir, entry.Name()))
			if err != nil {
				return false, err
			}
			if !empty {
				allEmpty = false
			}
		} else {
			allEmpty = false
		}
	}

	// Remove current directory if all entries were empty subdirectories (now removed)
	if allEmpty && dir != c.RepoRoot { // Avoid removing the repo root
		if !c.DryRun {
			fmt.Printf("[Removing Empty Dir]: %s\n", dir)
			err := os.Remove(dir)
			return err == nil, err
		}
		fmt.Printf("[Would Remove Empty Dir]: %s\n", dir)
		return true, nil
	}

	return false, nil
}

// removeSpecificFilesFromRoot handles the removal of specific files from the library root.
func (c *CleanCommand) removeSpecificFilesFromRoot(libraryPath string) int {
	filesToRemove := []string{}
	if c.additionalFiles != "" {
		additional := strings.Split(c.additionalFiles, ",")
		for _, file := range additional {
			trimmed := strings.TrimSpace(file)
			if trimmed != "" {
				filesToRemove = append(filesToRemove, trimmed)
			}
		}
	}

	removedCount := 0
	fmt.Println("Processing specific files in library root...")
	for _, file := range filesToRemove {
		filePath := filepath.Join(libraryPath, file)
		action := "Would remove"
		if !c.DryRun {
			action = "Removing"
			err := os.Remove(filePath)
			if err == nil {
				removedCount++
				fmt.Printf("[%s]: %s\n", action, filePath)
			} else if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error removing '%s': %v\n", filePath, err)
			} else {
				fmt.Printf("[Skipped (Not Found)]: %s\n", filePath)
			}
		} else {
			fmt.Printf("[%s]: %s\n", action, filePath)
		}
	}
	fmt.Printf("Specific file processing complete. Total files processed: %d\n", len(filesToRemove))
	if !c.DryRun {
		fmt.Printf("Total specific files actually removed: %d\n", removedCount)
	}
	return removedCount
}

// this is the entry point of the program
func main() {
	cleanCmd := NewCleanCommand()

	// Call execute with the arguments *after* the command name.
	if err := cleanCmd.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
