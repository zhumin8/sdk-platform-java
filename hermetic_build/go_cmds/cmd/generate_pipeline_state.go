package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/sdk-platform-java/hermetic_build/utils/internal/helpers"
	"github.com/googleapis/sdk-platform-java/hermetic_build/utils/internal/statepb"
	"gopkg.in/yaml.v3"
)

// Define the expected subdirectory for output within the input repo
const generationInputSubdir = "generation-input"

// Define the name of the output file
const pipelineStateFileName = "pipeline-state.json"

// GeneratePipelineStateCommand defines the structure for the generate-pipeline-state command.
type GeneratePipelineStateCommand struct {
	InputRepoPath string // Path to the root of the input repository

	flagSet *flag.FlagSet // To hold the flags for this command
}

// NewGeneratePipelineStateCommand creates a new GeneratePipelineStateCommand instance.
func NewGeneratePipelineStateCommand() *GeneratePipelineStateCommand {
	cmd := &GeneratePipelineStateCommand{
		flagSet: flag.NewFlagSet("generate-pipeline-state", flag.ExitOnError), // Initialize the flag set
	}
	cmd.initFlags() // Define flags
	return cmd
}

// initFlags defines the flags for the generate-pipeline-state command.
func (c *GeneratePipelineStateCommand) initFlags() {
	// Required flag: the path to the input repository
	c.flagSet.StringVar(&c.InputRepoPath, "input-repo-path", "", "Required: Path to the root of the input repository")

	// Set custom usage function for this flag set.
	c.flagSet.Usage = func() {
		// Assuming os.Args[0] is the executable name (like 'mycli' or 'utils')
		fmt.Fprintf(os.Stderr, "Usage: %s %s [options]\n", os.Args[0], c.flagSet.Name())
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Description:")
		fmt.Fprintln(os.Stderr, "  Reads generation configuration and versions, then constructs a pipeline-state.json file.")
		fmt.Fprintln(os.Stderr, "  The output file is written to a specific subdirectory within the input repository.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		c.flagSet.PrintDefaults()
	}
}

func getSubstringAfterColon(str string) string {
	index := strings.Index(str, ":")
	if index == -1 {
		return ""
	}
	return str[index+1:]
}

func convertGapicsToApiPaths(gapicConfigs []helpers.Gapics) []string {
	if gapicConfigs == nil {
		return []string{} // Return an empty slice if input is nil
	}
	apiPaths := make([]string, len(gapicConfigs))
	for i, gapic := range gapicConfigs {
		apiPaths[i] = gapic.ProtoPaths
	}
	return apiPaths
}

func readAndUnmarshalConfig(configFilePath string) (*helpers.RepoConfig, error) {
	fmt.Println("Reading and unmarshaling generation_config.yaml...")

	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		// Return a wrapped error with context
		return nil, fmt.Errorf("error reading config file %s: %w", configFilePath, err)
	}

	var repoConfig helpers.RepoConfig             // Declare a variable of RepoConfig type
	err = yaml.Unmarshal(configData, &repoConfig) // Unmarshal into the address of repoConfig
	if err != nil {
		// Return a wrapped error with context
		return nil, fmt.Errorf("error unmarshaling config file %s: %w", configFilePath, err)
	}
	fmt.Println("generation_config.yaml unmarshaled.")

	// Apply default values to the parsed configuration
	repoConfig.ApplyDefaults() // Call the SetDefaults method on the repoConfig instance
	fmt.Println("Defaults applied to generation_config.yaml data.")

	// Return the address of the populated and defaulted repoConfig
	return &repoConfig, nil
}

func buildPipelineState(
	repoConfig *helpers.RepoConfig,
	versions map[string]helpers.VersionInfo,
) (*statepb.PipelineState, error) {
	fmt.Println("Constructing pipeline state data structure using Protobuf structs...")

	pipelineState := &statepb.PipelineState{}

	// Populate top-level fields from repoConfig or with defaults
	pipelineState.ImageTag = "latest"

	// Populate the Libraries slice by iterating through repoConfig.Libraries
	// Pre-allocate capacity for efficiency
	pipelineState.Libraries = make([]*statepb.LibraryState, 0, len(repoConfig.Libraries))
	for _, libConfig := range repoConfig.Libraries {
		libraryState := &statepb.LibraryState{} // Create an instance for each library

		// Validate and populate libraryState.Id from LibraryName
		if libConfig.LibraryName == nil || *libConfig.LibraryName == "" {
			return nil, fmt.Errorf("LibraryName is required but missing or empty in config for api_shortname: %s", libConfig.APIShortname)
		}
		libraryState.Id = "java-" + *libConfig.LibraryName

		// Validate and extract artifact_id from DistributionName for versions lookup
		if libConfig.DistributionName == nil || *libConfig.DistributionName == "" {
			return nil, fmt.Errorf("DistributionName is required but missing or empty in config for library: %s", *libConfig.LibraryName)
		}
		artifactID := getSubstringAfterColon(*libConfig.DistributionName)
		if artifactID == "" {
			return nil, fmt.Errorf("failed to extract artifact_id from DistributionName '%s' for library: %s", *libConfig.DistributionName, *libConfig.LibraryName)
		}

		// Retrieve version info from the 'versions' map
		versionInfo, ok := versions[artifactID]
		if !ok {
			// This indicates a library in config.yaml does not have a corresponding entry in versions.txt.
			// Decide whether to treat this as an error or use default/empty versions.
			// For now, we'll return an error to highlight the missing data.
			return nil, fmt.Errorf("version info not found for artifact_id '%s' (library: %s) in versions.txt. "+
				"Please ensure versions.txt is complete or handle new libraries", artifactID, *libConfig.LibraryName)
		}

		libraryState.CurrentVersion = versionInfo.ReleasedVersion
		libraryState.NextVersion = strings.SplitN(versionInfo.CurrentVersion, "-", 2)[0]

		// Automation levels (hardcoded for now)
		libraryState.GenerationAutomationLevel = statepb.AutomationLevel_AUTOMATION_LEVEL_AUTOMATIC
		libraryState.ReleaseAutomationLevel = statepb.AutomationLevel_AUTOMATION_LEVEL_AUTOMATIC

		// Populate ApiPaths from Gapics using the helper
		libraryState.ApiPaths = convertGapicsToApiPaths(libConfig.Gapics)

		// SourcePaths is derived from Id
		libraryState.SourcePaths = []string{libraryState.Id}

		// Last Generated/Released Commit (from repoConfig.GoogleapisCommitish pointer)
		libraryState.LastGeneratedCommit = repoConfig.GoogleapisCommitish
		libraryState.LastReleasedCommit = repoConfig.GoogleapisCommitish // Assume same for now

		libraryState.ReleaseTimestamp = nil // Default: nil pointer for Timestamp

		pipelineState.Libraries = append(pipelineState.Libraries, libraryState)
	}
	fmt.Println("Pipeline state data structure constructed.")
	return pipelineState, nil
}

// Execute runs the generate-pipeline-state command.
func (c *GeneratePipelineStateCommand) Execute(args []string) error {
	fmt.Println("Executing generate-pipeline-state command...")

	// --- Parse Flags ---
	if err := c.flagSet.Parse(args); err != nil {
		return err // Handles --help and parsing errors
	}

	// --- Validate Required Flags ---
	if c.InputRepoPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -input-repo-path is required.\n")
		c.flagSet.Usage() // Show usage for this command
		return fmt.Errorf("required flag -input-repo-path not set")
	}

	// --- Validate Input Repository Path ---
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

	// --- Determine File Paths ---
	configFilePath := filepath.Join(c.InputRepoPath, "generation_config.yaml")
	versionsFilePath := filepath.Join(c.InputRepoPath, "versions.txt")
	outputDirPath := filepath.Join(c.InputRepoPath, generationInputSubdir) // The subdir for output
	outputFilePath := filepath.Join(outputDirPath, pipelineStateFileName)  // The final output file path

	fmt.Printf("Input Repository: %s\n", c.InputRepoPath)
	fmt.Printf("Reading config from: %s\n", configFilePath)
	fmt.Printf("Reading versions from: %s\n", versionsFilePath)
	fmt.Printf("Output directory: %s\n", outputDirPath)
	fmt.Printf("Writing pipeline state to: %s\n", outputFilePath)

	// --- Ensure Output Directory Exists ---
	// Create the output subdirectory if it doesn't exist
	err = os.MkdirAll(outputDirPath, 0755) // Use 0755 for typical directory permissions
	if err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDirPath, err)
	}

	// --- Start Main Logic ---
	// 1. Read and Unmarshal generation_config.yaml
	repoConfig, err := readAndUnmarshalConfig(configFilePath)
	if err != nil {
		return err
	}

	// 2. Read versions.txt
	fmt.Println("Reading versions.txt...")
	versions, err := helpers.LoadVersionsFromFile(versionsFilePath)
	if err != nil {
		return fmt.Errorf("error loading versions: %v", err)
	}
	fmt.Println("\nSuccessfully loaded versions.")

	// 3. Construct the data structure for pipeline-state.json
	pipelineState, err := buildPipelineState(repoConfig, versions)
	if err != nil {
		return fmt.Errorf("failed to construct pipeline state: %w", err)
	}

	// 4. Marshal the data structure to JSON
	fmt.Println("Marshaling pipeline state to JSON...")
	// Use json.MarshalIndent for a nicely formatted, human-readable JSON output
	// The generated Protobuf structs have JSON tags, so this works seamlessly.
	jsonData, err := json.MarshalIndent(pipelineState, "", "  ") // "" prefix, "  " indent
	if err != nil {
		return fmt.Errorf("error marshaling pipeline state to JSON: %w", err)
	}
	fmt.Println("Pipeline state marshaled to JSON.")

	// 5. Write the JSON data to the output file
	fmt.Printf("Writing pipeline-state.json to %s...\n", outputFilePath)
	// os.WriteFile is a convenient way to write the whole content
	err = os.WriteFile(outputFilePath, jsonData, 0644) // Use 0644 for typical file permissions
	if err != nil {
		return fmt.Errorf("error writing pipeline state file %s: %w", outputFilePath, err)
	}
	fmt.Println("pipeline-state.json written successfully.")

	fmt.Println("\nGenerate-pipeline-state command finished.")
	return nil
}
