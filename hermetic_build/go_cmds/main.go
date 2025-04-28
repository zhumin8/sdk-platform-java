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
	DeepRemoveRegex []string `yaml:"deep-remove-regex"`
}

// CleanCommand defines the structure for the clean command.
type CleanCommand struct {
	RepoRoot  string
	LibraryID string
	DryRun    bool
	flagSet   *flag.FlagSet // To hold the flags for this command
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

	//  Add cleaning logic here.

	libraryPath := filepath.Join(c.RepoRoot, c.LibraryID)

	config, err := c.loadOwlBotConfig(libraryPath)
	if err != nil {
		return err
	}

	fmt.Printf("Repository Root: %s\n", c.RepoRoot)
	fmt.Printf("Library ID: %s\n", c.LibraryID)
	fmt.Printf("Dry-run mode: %v\n", c.DryRun)

	if len(config.DeepRemoveRegex) > 0 {
		fmt.Println("Processing deep-remove-regex...")
		if err := c.performRegexRemoval(libraryPath, c.LibraryID, config.DeepRemoveRegex); err != nil {
			return err
		}
		fmt.Println("Deep removal process complete.")
	} else {
		fmt.Println("No deep-remove-regex found in .OwlBot-hermetic.yaml. Nothing to remove.")
	}

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

// performRegexRemoval reads deep-remove-regex and removes matching paths within the libraryPath.
func (c *CleanCommand) performRegexRemoval(libraryPath string, libraryID string, regexes []string) error {
	matches := []string{}

	for _, pattern := range regexes {
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

		re, err := regexp.Compile(relativePattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Invalid regex '%s' (relative '%s'): %v\n", pattern, relativePattern, err)
			continue
		}

		err = filepath.Walk(libraryPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == libraryPath {
				return nil // Skip the libraryPath itself
			}

			relativePath := filepath.ToSlash(filepath.Clean(path[len(libraryPath)+1:]))

			if re.MatchString(relativePath) {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error during regex matching for pattern '%s': %w", pattern, err)
		}
	}

	// Sort the matches in reverse order (deepest first)
	sort.Slice(matches, func(i, j int) bool {
		return len(strings.Split(matches[i], string(filepath.Separator))) > len(strings.Split(matches[j], string(filepath.Separator)))
	})

	// Remove the matched paths
	for _, pathToRemove := range matches {
		action := "Would remove"
		if !c.DryRun {
			action = "Removing"
			err := os.RemoveAll(pathToRemove)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error removing '%s': %v\n", pathToRemove, err)
			}
		}
		fmt.Printf("[%s]: %s\n", action, pathToRemove)
	}

	fmt.Println("Deep removal process complete.")
	return nil
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
