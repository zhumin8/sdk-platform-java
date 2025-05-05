package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Command interface defines the common methods for all commands.
type Command interface {
	Execute(args []string) error
	// Name() string // Method to get the command name
	// Usage() // Method to print command usage
}

// Helper to print usage
func printUsage() {
	fmt.Println("Usage: mycli <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("  clean         Perform cleaning tasks")
	fmt.Println("  build-library Build the library")
}

// Define the path to your Python entrypoint script within the Docker image
const defaultPythonEntrypointScript = "/src/library_generation/cli/entry_point.py"
const localPythonPathEnvVar = "LOCAL_PYTHON_SCRIPT_PATH"

// Function to get the actual script path, allowing override for local testing
func getPythonEntrypointPath() string {
	// Check for the environment variable
	localPath := os.Getenv(localPythonPathEnvVar)
	if localPath != "" {
		// If the environment variable is set, use its value
		fmt.Fprintf(os.Stderr, "Using local Python script path from env (%s): %s\n", localPythonPathEnvVar, localPath)
		return localPath
	}
	// If the environment variable is NOT set, use the default path (for Docker)
	fmt.Fprintf(os.Stderr, "Using default Python script path: %s\n", defaultPythonEntrypointScript)
	return defaultPythonEntrypointScript
}

func main() {
	// Check minimum arguments
	if len(os.Args) < 2 {
		fmt.Println("Error: No command provided.")
		printUsage()
		os.Exit(1)
	}

	// Get the command name
	commandName := os.Args[1]
	argsForCommand := os.Args[2:] // Correctly get arguments *after* the command name

	var commandToExecute Command // Use the Command interface

	switch commandName {
	case "clean":
		commandToExecute = NewCleanCommand()
	case "build-library":
		commandToExecute = NewBuildLibraryCommand()
	case "generate":
		// Execute the Python script for the 'generate' command

		// We need to build the command line for 'python' like this:
		// python /src/path/to/script generate --flag1 value --flag2 value

		// os.Args[1:] contains "generate" and all the arguments that came after it on the command line
		// These are the arguments we want to pass *to* the Python script.
		argsForPythonScript := os.Args[1:]

		// The command to execute is "python"
		// Its arguments start with the script path, followed by the argsForPythonScript slice.
		// We use append to create the full list of arguments for the 'python' executable.
		pythonScriptPath := getPythonEntrypointPath()
		pythonCmdArgs := append([]string{pythonScriptPath}, argsForPythonScript...)

		fmt.Printf("Dispatching to Python: python %s\n", strings.Join(pythonCmdArgs, " "))

		cmd := exec.Command("python", pythonCmdArgs...) // Use os/exec to run Python

		// Connect Python's stdout/stderr so the user sees its output
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing python command '%s': %v\n", commandName, err)
			// Check if it's an ExitError to get the exit code if needed
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			} else {
				os.Exit(1) // Generic error exit code
			}
		}
		// If Python script ran successfully, it should exit 0 itself.
		// No need for os.Exit(0) here as the Go program will simply exit 0.

	default:
		fmt.Printf("Error: Unknown command '%s'\n", commandName)
		printUsage()
		os.Exit(1)
	}

	// Execute the selected command and check for errors consistently
	if err := commandToExecute.Execute(argsForCommand); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command '%s': %v\n", commandName, err)
		os.Exit(1)
	}

	// Explicitly exit with success status
	os.Exit(0)
}
