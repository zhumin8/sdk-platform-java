package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BuildLibraryCommand struct {
	RepoRoot  string
	LibraryID string
	Test      bool
	flagSet   *flag.FlagSet
}

func NewBuildLibraryCommand() *BuildLibraryCommand {
	cmd := &BuildLibraryCommand{
		flagSet: flag.NewFlagSet("build-library", flag.ExitOnError),
	}
	cmd.initFlags()
	return cmd
}

func (b *BuildLibraryCommand) initFlags() {
	b.flagSet.StringVar(&b.RepoRoot, "repo-root", "", "Path to the root of the language repo; required")
	b.flagSet.StringVar(&b.LibraryID, "library-id", "", "ID of the library to build, e.g., java-accesscontextmanager")
	b.flagSet.BoolVar(&b.Test, "test", false, "Run unit tests when building.")

	// Set custom usage function for this flag set.
	b.flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s [options]\n", os.Args[0], b.flagSet.Name())
		fmt.Fprintln(os.Stderr, "Options:")
		b.flagSet.PrintDefaults()
	}
}

// Define the Execute method for BuildLibraryCommand
func (b *BuildLibraryCommand) Execute(args []string) error {

	if err := b.flagSet.Parse(args); err != nil {
		return err
	}

	if b.RepoRoot == "" {
		fmt.Println("Error: -repo-root is required.")
		b.flagSet.Usage()
		return fmt.Errorf("required flags not set")
	}
	if b.LibraryID == "" {
		fmt.Println("Omitting -library-id is not implemented yet.")
		b.flagSet.Usage()
		return fmt.Errorf("required flags not set")
	}

	fmt.Println("Executing build-library command...")
	libraryPath := filepath.Join(b.RepoRoot, b.LibraryID)

	fmt.Printf("Attempting to build library '%s'...\n", b.LibraryID)
	libraryMavenArgs := []string{"install", "-Dcheckstyle.skip=true"}
	if !b.Test {
		libraryMavenArgs = append(libraryMavenArgs, "-DskipUnitTests=true")
	}
	if err := RunMavenCommandInDir(libraryPath, libraryMavenArgs); err != nil {
		return fmt.Errorf("failed to install library %s: %w", b.LibraryID, err)
	}

	fmt.Println("Library install finished.")

	fmt.Println("Build-library command finished successfully.")
	return nil
}

// RunMavenCommandInDir executes the mvn command with the given arguments in the specified directory.
// It streams stdout and stderr to the console and returns an error if the command fails.
func RunMavenCommandInDir(dir string, mavenArgs []string) error {
	// Check if the directory exists and is a directory (good practice)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dir)
		}
		return fmt.Errorf("error checking directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dir)
	}

	// Define the command to run: "mvn <arg1> <arg2> ..."
	// Pass the base command "mvn" and then all arguments from the slice
	cmd := exec.Command("mvn", mavenArgs...)

	// Set the working directory for the command
	cmd.Dir = dir

	// Connect the command's standard output and error streams to the main process's streams
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running command: mvn %s in %s\n", strings.Join(mavenArgs, " "), dir)

	// Run the command and wait for it to complete.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mvn command failed in %s: %w", dir, err)
	}

	return nil
}
