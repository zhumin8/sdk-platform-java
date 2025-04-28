package main

import (
	"fmt"
	"os"
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
