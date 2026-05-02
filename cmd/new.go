package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// Resource defines the common fields for all nocti resources
type Resource struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	CreatedAt string  `json:"created_at"`
	Parent    *Parent `json:"parent,omitempty"`
}

type Parent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Notebook defines the structure of a notebook entry
type Notebook Resource

// Todo defines the structure of a todo list entry
type Todo Resource

// Calendar defines the structure of a calendar entry
type Calendar Resource

type ColorsConfig struct {
	FileList     string `json:"file_list,omitempty"`
	PreviewPane  string `json:"preview_pane,omitempty"`
	HelpBg       string `json:"help_bg,omitempty"`
	HelpFg       string `json:"help_fg,omitempty"`
	HelpBorderBg string `json:"help_border_bg,omitempty"`
	HelpBorderFg string `json:"help_border_fg,omitempty"`
}

// FullConfig to include all resource types
type FullConfig struct {
	Name      string        `json:"name"`
	Version   string        `json:"version"`
	Editor    string        `json:"editor,omitempty"`
	Colors    *ColorsConfig `json:"colors,omitempty"`
	Notebooks []Notebook    `json:"notebooks,omitempty"`
	Todos     []Todo        `json:"todos,omitempty"`
	Calendars []Calendar    `json:"calendars,omitempty"`
}

func GenerateID() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b)
}

var NewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new resource",
	Long: `Create a new resource like a notebook, todo list, or calendar.
Resources are stored in the .nocti/nocti.json file in your project directory.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := FindProjectRoot()
		if err != nil {
			return fmt.Errorf("you need to init with `nocti init` before creating new resources (error: %v)", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var choice string
		prompt := &survey.Select{
			Message: "What would you like to create?",
			Options: []string{"notebook", "todo", "calendar"},
		}

		err := survey.AskOne(prompt, &choice)
		if err != nil {
			return err
		}

		switch choice {
		case "notebook":
			return newNotebookCmd.RunE(newNotebookCmd, args)
		case "todo":
			return newTodoCmd.RunE(newTodoCmd, args)
		case "calendar":
			return newCalendarCmd.RunE(newCalendarCmd, args)
		}

		return nil
	},
}

func FindProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, ".nocti", "nocti.json")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", fmt.Errorf(".nocti/nocti.json not found")
}

var ResourceName string
var Overwrite bool

func CreateResource(resourceType string) error {
	root, err := FindProjectRoot()
	if err != nil {
		return err
	}
	filename := filepath.Join(root, ".nocti/nocti.json")

	// Detect parent resource if inside one
	var parentID string
	var parentName string
	if _, err := os.Stat(".nocti.json"); err == nil {
		localData, err := os.ReadFile(".nocti.json")
		if err == nil {
			var localConfig map[string]interface{}
			if err := json.Unmarshal(localData, &localConfig); err == nil {
				if id, ok := localConfig["id"].(string); ok {
					parentID = id
				}
				if name, ok := localConfig["name"].(string); ok {
					parentName = name
				}
			}
		}
	}

	// Use flag if provided, otherwise prompt
	name := ResourceName
	if name == "" {
		prompt := &survey.Input{
			Message: fmt.Sprintf("Enter %s name:", resourceType),
		}
		err := survey.AskOne(prompt, &name, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
	}
	name = strings.TrimSpace(name)

	// Read existing config
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config FullConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	var res Resource
	var skipConfigUpdate bool

	// Check if a resource with this name already exists
	var existingResources []Resource
	switch resourceType {
	case "notebook":
		for _, r := range config.Notebooks {
			existingResources = append(existingResources, Resource(r))
		}
	case "todo":
		for _, r := range config.Todos {
			existingResources = append(existingResources, Resource(r))
		}
	case "calendar":
		for _, r := range config.Calendars {
			existingResources = append(existingResources, Resource(r))
		}
	}

	for _, r := range existingResources {
		if r.Name == name {
			res = r
			skipConfigUpdate = true
			break
		}
	}

	if !skipConfigUpdate {
		// Check if folder already exists for a new resource
		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("folder '%s' already exists and is not registered as a %s; %s names must be unique", name, resourceType, resourceType)
		}

		// Create a map of all existing IDs to ensure uniqueness across all types
		existingIDs := make(map[string]bool)
		for _, nb := range config.Notebooks {
			existingIDs[nb.ID] = true
		}
		for _, t := range config.Todos {
			existingIDs[t.ID] = true
		}
		for _, c := range config.Calendars {
			existingIDs[c.ID] = true
		}

		// Generate a unique ID
		newID := GenerateID()
		for existingIDs[newID] {
			newID = GenerateID()
		}

		res = Resource{
			ID:        newID,
			Name:      name,
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		if parentID != "" {
			res.Parent = &Parent{
				ID:   parentID,
				Name: parentName,
			}
		}

		// Add to correct slice
		switch resourceType {
		case "notebook":
			config.Notebooks = append(config.Notebooks, Notebook(res))
		case "todo":
			config.Todos = append(config.Todos, Todo(res))
		case "calendar":
			config.Calendars = append(config.Calendars, Calendar(res))
		}

		// Save updated config
		updatedData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal updated config: %w", err)
		}

		if err := os.WriteFile(filename, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Handle folder and hidden .nocti.json for all resource types
	// Create directory if it doesn't exist
	if _, err := os.Stat(name); os.IsNotExist(err) {
		if err := os.Mkdir(name, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", name, err)
		}
	}

	// Create or update .nocti.json inside the resource folder
	resourceConfigPath := filepath.Join(name, ".nocti.json")
	if _, err := os.Stat(resourceConfigPath); err == nil && !Overwrite {
		return fmt.Errorf("file %s already exists and will not be overwritten (use -o to overwrite)", resourceConfigPath)
	}

	editor := config.Editor
	if editor == "" {
		editor = "nano"
	}

	resInfo := map[string]interface{}{
		"id":         res.ID,
		"name":       res.Name,
		"type":       resourceType,
		"created_at": res.CreatedAt,
		"editor":     editor,
	}

	if resourceType == "notebook" {
		resInfo["colors"] = &ColorsConfig{
			FileList:     "blue",
			PreviewPane:  "orange",
			HelpBg:       "darkgray",
			HelpFg:       "white",
			HelpBorderBg: "black",
			HelpBorderFg: "gray",
		}
	}

	if res.Parent != nil {
		resInfo["parent"] = map[string]string{
			"id":   res.Parent.ID,
			"name": res.Parent.Name,
		}
	}

	// If we are inside another resource, update its .nocti.json
	if _, err := os.Stat(".nocti.json"); err == nil {
		localData, err := os.ReadFile(".nocti.json")
		if err == nil {
			var localConfig map[string]interface{}
			if err := json.Unmarshal(localData, &localConfig); err == nil {
				resources, ok := localConfig["resources"].([]interface{})
				if !ok {
					resources = []interface{}{}
				}

				// Check if already in list (e.g. if re-creating)
				found := false
				for i, r := range resources {
					if rm, ok := r.(map[string]interface{}); ok && rm["id"] == res.ID {
						resources[i] = resInfo
						found = true
						break
					}
				}
				if !found {
					resources = append(resources, resInfo)
				}
				localConfig["resources"] = resources

				updatedLocalData, err := json.MarshalIndent(localConfig, "", "  ")
				if err == nil {
					os.WriteFile(".nocti.json", updatedLocalData, 0644)
				}
			}
		}
	}

	metadataData, err := json.MarshalIndent(resInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s metadata: %w", resourceType, err)
	}

	if err := os.WriteFile(resourceConfigPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write %s metadata: %w", resourceType, err)
	}

	fmt.Printf("Successfully created %s: %s (ID: %s)\n", resourceType, name, res.ID)
	return nil
}

var newNotebookCmd = &cobra.Command{
	Use:   "notebook",
	Short: "Create a new notebook",
	RunE: func(cmd *cobra.Command, args []string) error {
		return CreateResource("notebook")
	},
}

var newTodoCmd = &cobra.Command{
	Use:   "todo",
	Short: "Create a new todo list",
	RunE: func(cmd *cobra.Command, args []string) error {
		return CreateResource("todo")
	},
}

var newCalendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Create a new calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		return CreateResource("calendar")
	},
}

func init() {
	NewCmd.PersistentFlags().StringVarP(&ResourceName, "name", "n", "", "Name of the resource to create")
	NewCmd.PersistentFlags().BoolVarP(&Overwrite, "overwrite", "o", false, "Overwrite existing .nocti.json file in notebook directory")
	NewCmd.AddCommand(newNotebookCmd)
	NewCmd.AddCommand(newTodoCmd)
	NewCmd.AddCommand(newCalendarCmd)
	RootCmd.AddCommand(NewCmd)
}
