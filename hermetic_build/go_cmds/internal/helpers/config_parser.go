package helpers

import (
	"fmt"
	"strings"
	// Imports for YAML, JSON, and data structures will be added later
	// "encoding/json"
)

// --- Structs for generation_config.yaml Parsing ---

type RepoConfig struct {
	GoogleapisCommitish   string          `yaml:"googleapis_commitish"`
	LibrariesBomVersion   *string         `yaml:"libraries_bom_version"`
	GapicGeneratorVersion *string         `yaml:"gapic_generator_version"`
	Libraries             []LibraryConfig `yaml:"libraries"`
}

type LibraryConfig struct {
	APIShortname         string   `yaml:"api_shortname"`
	APIDescription       string   `yaml:"api_description"`
	NamePretty           string   `yaml:"name_pretty"`
	ProductDocs          string   `yaml:"product_documentation"`
	LibraryType          *string  `yaml:"library_type"`
	ReleaseLevel         *string  `yaml:"release_level"`
	APIID                *string  `yaml:"api_id"`
	APIReference         *string  `yaml:"api_reference"`
	CodeownerTeam        *string  `yaml:"codeowner_team"`
	ClientDocumentation  *string  `yaml:"client_documentation"`
	DistributionName     *string  `yaml:"distribution_name"`
	ExcludedPoms         *string  `yaml:"excluded_poms"`
	ExcludedDependencies *string  `yaml:"excluded_dependencies"`
	GoogleapisCommitish  *string  `yaml:"googleapis_commitish"`
	GroupID              *string  `yaml:"group_id"`
	IssueTracker         *string  `yaml:"issue_tracker"`
	LibraryName          *string  `yaml:"library_name"`
	RESTDocumentation    *string  `yaml:"rest_documentation"`
	RPCDocumentation     *string  `yaml:"rpc_documentation"`
	CloudAPI             *bool    `yaml:"cloud_api"`
	RequiresBilling      *bool    `yaml:"requires_billing"`
	Transport            *string  `yaml:"transport"`
	Gapics               []Gapics `yaml:"GAPICs"`
}

type Gapics struct {
	ProtoPaths string `yaml:"proto_path"`
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
func boolPtr(b bool) *bool {
	return &b
}

// ApplyDefaults iterates through libraries and sets default values for optional fields.
func (repoConfig *RepoConfig) ApplyDefaults() {
	if repoConfig == nil {
		return
	}

	for i := range repoConfig.Libraries {
		lib := &repoConfig.Libraries[i] // Get a pointer to the library to modify it in place

		if lib.ReleaseLevel == nil {
			fmt.Printf("Applying default ReleaseLevel 'preview' to library '%s'\n", lib.APIShortname)
			lib.ReleaseLevel = stringPtr("preview")
		}
		if lib.LibraryType == nil {
			fmt.Printf("Applying default LibraryType 'GAPIC_AUTO' to library '%s'\n", lib.APIShortname)
			lib.LibraryType = stringPtr("GAPIC_AUTO")
		}
		if lib.APIID == nil {
			if lib.APIShortname != "" {
				defaultApiId := fmt.Sprintf("%s.googleapis.com", strings.ToLower(lib.APIShortname))
				fmt.Printf("Applying default APIID '%s' to library '%s'\n", defaultApiId, lib.APIShortname)
				lib.APIID = stringPtr(defaultApiId)
			} else {
				fmt.Printf("Warning: Cannot set default APIID for library with empty APIShortname (index %d)\n", i)
			}
		}
		if lib.GroupID == nil {
			fmt.Printf("Applying default GroupID 'com.google.cloud' to library '%s'\n", lib.APIShortname)
			lib.GroupID = stringPtr("com.google.cloud")
		}

		if lib.LibraryName == nil {
			if lib.APIShortname != "" {
				fmt.Printf("Applying default LibraryName '%s' to library '%s'\n", lib.APIShortname, lib.APIShortname)
				lib.LibraryName = stringPtr(lib.APIShortname)
			} else {
				fmt.Printf("Warning: Cannot set default LibraryName for library with empty APIShortname (index %d)\n", i)
			}
		}

		if lib.CloudAPI == nil {
			fmt.Printf("Applying default CloudAPI 'true' to library '%s'\n", lib.APIShortname)
			lib.CloudAPI = boolPtr(true)
		}
		if lib.RequiresBilling == nil {
			fmt.Printf("Applying default RequiresBilling 'true' to library '%s'\n", lib.APIShortname)
			lib.RequiresBilling = boolPtr(true)
		}

		// --- Now, apply default for DistributionName (depends on the above) ---
		if lib.DistributionName == nil {
			// Get GroupID (should be non-nil here due to above default)
			groupIDValue := *lib.GroupID

			// Determine cloudPrefix (CloudAPI should be non-nil here due to above default)
			cloudPrefix := ""
			if lib.CloudAPI != nil && *lib.CloudAPI { // Check if CloudAPI is true
				cloudPrefix = "cloud-"
			}

			// Get LibraryName (LibraryName should be non-nil if APIShortname was present)
			libraryNameValue := ""
			if lib.LibraryName != nil {
				libraryNameValue = *lib.LibraryName
			}

			// Only construct if essential components (like library_name) are present
			// Or, decide on behavior if libraryNameValue is empty.
			// The Python code would proceed even with an empty library_name.
			if groupIDValue == "" && libraryNameValue == "" {
				fmt.Printf("Skipping default DistributionName for library '%s' due to missing GroupID and LibraryName\n", lib.APIShortname)
			} else {
				defaultDistName := fmt.Sprintf("%s:google-%s%s", groupIDValue, cloudPrefix, libraryNameValue)
				fmt.Printf("Applying default DistributionName '%s' to library '%s'\n", defaultDistName, lib.APIShortname)
				lib.DistributionName = stringPtr(defaultDistName)
			}
		}
		// Uniqueness check for LibraryName (optional, can be expanded)
		// if lib.LibraryName != nil {
		// 	if _, exists := libraryNameSet[*lib.LibraryName]; exists {
		// 		fmt.Printf("Warning: LibraryName '%s' for APIShortname '%s' is not unique.\n", *lib.LibraryName, lib.APIShortname)
		// 	}
		// 	libraryNameSet[*lib.LibraryName] = true
		// }
	}
}

// --- End Structs for generation_config.yaml Parsing ---
