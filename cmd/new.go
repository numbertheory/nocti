package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// Resource defines the common fields for all nocti resources
type Resource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Notebook defines the structure of a notebook entry
type Notebook Resource

// Todo defines the structure of a todo list entry
type Todo Resource

// Calendar defines the structure of a calendar entry
type Calendar Resource

// FullConfig to include all resource types
type FullConfig struct {
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Notebooks []Notebook `json:"notebooks,omitempty"`
	Todos     []Todo     `json:"todos,omitempty"`
	Calendars []Calendar `json:"calendars,omitempty"`
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
		if _, err := os.Stat(".nocti/nocti.json"); os.IsNotExist(err) {
			return fmt.Errorf("you need to init with `nocti init` before creating new resources")
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

var ResourceName string

func CreateResource(resourceType string) error {
	filename := ".nocti/nocti.json"

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

	res := Resource{
		ID:        newID,
		Name:      name,
		CreatedAt: time.Now().Format(time.RFC3339),
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

	fmt.Printf("Successfully created %s: %s (ID: %s)\n", resourceType, name, newID)
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
	NewCmd.AddCommand(newNotebookCmd)
	NewCmd.AddCommand(newTodoCmd)
	NewCmd.AddCommand(newCalendarCmd)
	RootCmd.AddCommand(NewCmd)
}
