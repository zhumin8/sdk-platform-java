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
	// Usage() // Method to print command usage
}

// Helper to print usage
func printUsage() {
	fmt.Println("Usage: utils <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("Commands:")
	fmt.Println("  clean         Perform cleaning tasks (Go)")
	fmt.Println("  build-library Build the library (Go)")
	fmt.Println("  prep-input    Prepare input files for library generation (Go). This is intended be removed in the future")
	fmt.Println("  generate-pipeline-state Generate pipeline state file (Go)")
	fmt.Println("  generate      Generate libraries (Python)")
	fmt.Println("")
	fmt.Println("Use 'utils <command> --help' for command-specific options.")
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

// executeAndHandleError is a helper function to run a command's Execute method
// and handle the error if it occurs by printing to stderr and exiting.
func executeAndHandleError(
	commandName string, // Name of the command for error reporting
	commandToExecute interface{ Execute(args []string) error }, // The command instance implementing the interface
	argsForCommand []string, // Arguments to pass to the Execute method
) {
	// Execute the command's logic
	if err := commandToExecute.Execute(argsForCommand); err != nil {
		// If Execute returns an error, print it and exit
		fmt.Fprintf(os.Stderr, "Error executing command '%s': %v\n", commandName, err)
		os.Exit(1) // Exit with a non-zero status code to indicate failure
	}
	// If Execute returns nil, the function simply returns,
	// and the main function can proceed to os.Exit(0)
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
	handledInternally := false

	switch commandName {
	case "clean":
		commandToExecute = NewCleanCommand()
		executeAndHandleError(commandName, commandToExecute, argsForCommand)

	case "build-library":
		commandToExecute = NewBuildLibraryCommand()
		executeAndHandleError(commandName, commandToExecute, argsForCommand)
	case "prep-input":
		commandToExecute = NewPrepInputCommand()
		executeAndHandleError(commandName, commandToExecute, argsForCommand)
	case "generate-pipeline-state":
		commandToExecute = NewGeneratePipelineStateCommand()
		executeAndHandleError(commandName, commandToExecute, argsForCommand)

	case "generate":
		// Execute the Python script for the 'generate' command

		// We need to build the command line for 'python' like this:
		// python /src/path/to/script generate --flag1 value --flag2 value

		// os.Args[1:] contains "generate" and all the arguments that came after it on the command line
		// These are the arguments we want to pass *to* the Python script.
		argsForPythonScript := os.Args[1:]
		argsForPythonScript = append(argsForPythonScript, "--skip-gapic-bom=true")

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
		handledInternally = true
		// If Python script ran successfully, it should exit 0 itself.
		// No need for os.Exit(0) here as the Go program will simply exit 0.

	default:
		fmt.Printf("Error: Unknown command '%s'\n", commandName)
		printUsage()
		os.Exit(1)
	}

	if !handledInternally {
		os.Exit(0) // Exit with success status
	}
}
