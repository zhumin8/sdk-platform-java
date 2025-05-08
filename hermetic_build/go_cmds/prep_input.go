package main

import (
	"flag"
	"fmt"
	"io" // Needed for io.Copy
	"os"
	"path/filepath"
	"strings" // Needed for strings.HasPrefix
)

// PrepInputCommand defines the structure for the prep-input command.
type PrepInputCommand struct {
	InputRepoPath string
	OutputFolder  string
	flagSet       *flag.FlagSet // To hold the flags for this command
}

// NewPrepInputCommand creates a new PrepInputCommand instance.
func NewPrepInputCommand() *PrepInputCommand {
	cmd := &PrepInputCommand{
		flagSet: flag.NewFlagSet("prep-input", flag.ExitOnError), // Initialize the flag set
	}
	cmd.initFlags() // Define flags
	return cmd
}

// initFlags defines the flags for the prep-input command.
func (c *PrepInputCommand) initFlags() {
	// Required flags for this command
	c.flagSet.StringVar(&c.InputRepoPath, "input-repo-path", "", "Required: Path to the root of the input repository clone")
	c.flagSet.StringVar(&c.OutputFolder, "output-folder", "", "Required: Path to the output directory where files will be copied")

	// Set custom usage function for this flag set.
	c.flagSet.Usage = func() {
		// Assuming os.Args[0] is the executable name (like 'mycli' or 'utils')
		fmt.Fprintf(os.Stderr, "Usage: %s %s [options]\n", os.Args[0], c.flagSet.Name())
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Description:")
		fmt.Fprintln(os.Stderr, "  Copies specific configuration and metadata files from an input repository")
		fmt.Fprintln(os.Stderr, "  into a structured output directory.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		c.flagSet.PrintDefaults()
	}
}

// Execute runs the prep-input command.
func (c *PrepInputCommand) Execute(args []string) error {
	fmt.Println("Executing prep-input command...")

	// Parse the flags
	if err := c.flagSet.Parse(args); err != nil {
		return err // Handles --help and parsing errors
	}

	// Validate that the required flags are set after parsing.
	if c.InputRepoPath == "" || c.OutputFolder == "" {
		fmt.Fprintf(os.Stderr, "Error: Both -input-repo-path and -output-folder are required.\n")
		c.flagSet.Usage() // Show usage for this command
		return fmt.Errorf("required flags not set")
	}

	// --- Input Validation ---
	inputInfo, err := os.Stat(c.InputRepoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input repository path does not exist: %s", c.InputRepoPath)
		}
		return fmt.Errorf("error checking input repository path %s: %w", c.InputRepoPath, err)
	}
	if !inputInfo.IsDir() {
		return fmt.Errorf("input repository path is not a directory: %s", c.InputRepoPath)
	}

	// --- Ensure Output Directory Exists ---
	err = os.MkdirAll(c.OutputFolder, 0755) // Use 0755 for typical directory permissions
	if err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", c.OutputFolder, err)
	}

	fmt.Printf("Input Repository: %s\n", c.InputRepoPath)
	fmt.Printf("Output Folder: %s\n", c.OutputFolder)

	// --- Task 1: Copy specific files from the input root to the output root ---
	filesToCopyFromRoot := []string{"versions.txt", "generation_config.yaml"}
	fmt.Println("Copying specific files from input root...")
	for _, fileName := range filesToCopyFromRoot {
		srcPath := filepath.Join(c.InputRepoPath, fileName)
		destPath := filepath.Join(c.OutputFolder, fileName)
		err := copyFile(srcPath, destPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Warning: Source file not found, skipping: %s\n", srcPath)
			} else {
				return fmt.Errorf("error copying file %s to %s: %w", srcPath, destPath, err)
			}
		} else {
			fmt.Printf("Copied: %s to %s\n", srcPath, destPath)
		}
	}
	fmt.Println("Finished copying specific files from input root.")

	// --- Task 2: Process java-* subdirectories ---
	fmt.Println("\nProcessing java-* subdirectories...")
	entries, err := os.ReadDir(c.InputRepoPath)
	if err != nil {
		return fmt.Errorf("error reading input repository directory: %w", err)
	}

	for _, entry := range entries {
		// Check if it's a directory and its name starts with "java-"
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "java-") {
			javaDirName := entry.Name()
			javaSrcDirPath := filepath.Join(c.InputRepoPath, javaDirName)
			javaDestDirPath := filepath.Join(c.OutputFolder, javaDirName)

			// Ensure the destination subdirectory exists
			err := os.MkdirAll(javaDestDirPath, 0755)
			if err != nil {
				return fmt.Errorf("failed to create output subdirectory %s: %w", javaDestDirPath, err)
			}

			fmt.Printf("Processing subdirectory: %s\n", javaDirName)

			// --- Copy specific top-level files from this java-* directory ---
			// These are files expected *directly* inside the java-* directory
			topLevelFilesToCopy := []string{".OwlBot-hermetic.yaml", "clirr-ignored-differences.xml"}
			fmt.Println("  Copying specific top-level files...")
			for _, fileName := range topLevelFilesToCopy {
				srcPath := filepath.Join(javaSrcDirPath, fileName)
				destPath := filepath.Join(javaDestDirPath, fileName)
				fmt.Printf("  Attempting to copy: %s\n", fileName)
				err := copyFile(srcPath, destPath)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Printf("  Warning: Source file not found, skipping: %s\n", srcPath)
					} else {
						return fmt.Errorf("error copying top-level file %s to %s: %w", srcPath, destPath, err)
					}
				} else {
					fmt.Printf("  Copied: %s to %s\n", srcPath, destPath)
				}
			}

			// --- Recursively find and copy pom.xml files within this java-* directory ---
			fmt.Println("  Searching for and copying pom.xml files...")
			err = filepath.Walk(javaSrcDirPath, func(path string, info os.FileInfo, err error) error {
				// Handle errors encountered during the walk (e.g., permissions denied on a subdirectory)
				if err != nil {
					return fmt.Errorf("walk error in %s: %w", path, err)
				}

				// Skip src and test folders
				if info.IsDir() {
					if info.Name() == "src" {
						return filepath.SkipDir
					}
					if info.Name() == "test" {
						return filepath.SkipDir
					}
					return nil
				}

				if strings.ToLower(info.Name()) == "pom.xml" {
					// Calculate the relative path from the java-* source directory
					// This preserves the subdirectory structure within the java-* folder
					relativePath, err := filepath.Rel(javaSrcDirPath, path)
					if err != nil {
						// This shouldn't happen if path is inside javaSrcDirPath, but handle defensively
						fmt.Fprintf(os.Stderr, "Error calculating relative path for %s from %s: %v\n", path, javaSrcDirPath, err)
						return nil
					}

					destPath := filepath.Join(javaDestDirPath, relativePath)

					// Ensure the destination directory for the pom.xml exists (it might be nested)
					// filepath.Dir(destPath) gets the parent directory of the destination file
					destDir := filepath.Dir(destPath)
					if err := os.MkdirAll(destDir, 0755); err != nil {
						// This is a critical error for this specific pom.xml copy, stop the walk for this java-* dir
						return fmt.Errorf("failed to create destination directory %s for pom.xml: %w", destDir, err)
					}

					// Copy the pom.xml file
					fmt.Printf("  Attempting to copy pom.xml: %s (relative: %s)\n", info.Name(), relativePath)
					if err := copyFile(path, destPath); err != nil {
						// copyFile already handles os.IsNotExist as a warning internally.
						// If copyFile returns other errors, log and return the error to stop the walk for this java-* dir.
						// copyFile's error message is informative, so just wrap and return it.
						return fmt.Errorf("error copying pom.xml %s to %s: %w", path, destPath, err)
					} else {
						fmt.Printf("  Copied: %s to %s\n", path, destPath)
					}
				}

				// Return nil to continue walking to the next file or directory
				return nil
			})

			// Handle error returned by filepath.Walk. If walk returned an error, it's reported here.
			if err != nil {
				// Wrap the walk error with context about which java-* directory it occurred in
				return fmt.Errorf("error walking directory %s: %w", javaSrcDirPath, err)
			}
			fmt.Println("  Finished searching for and copying pom.xml files.")
		}
	}
	fmt.Println("Finished processing java-* subdirectories.")

	fmt.Println("\nPrep-input command finished successfully.")
	return nil // Indicate success
}

// copyFile is a helper function to copy a single file from src to dest.
// It returns an error if the copy fails.
func copyFile(srcPath, destPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close() // Ensure the source file is closed when the function exits

	// Create the destination file (this will overwrite if it exists)
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close() // Ensure the destination file is closed

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
