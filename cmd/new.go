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

// Notebook defines the structure of a notebook entry
type Notebook struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func generateID() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b)
}

// Updated Config to include notebooks
type FullConfig struct {
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Notebooks []Notebook `json:"notebooks,omitempty"`
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new resource",
	Long:  `Create a new resource like a notebook, todo, or calendar.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".nocti/nocti.json"); os.IsNotExist(err) {
			return fmt.Errorf("you need to init with `nocti init` before creating new resources")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a subcommand was provided, Cobra handles it.
		// If not, we show the interactive menu.
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
			fmt.Println("Todo creation not yet implemented")
		case "calendar":
			fmt.Println("Calendar creation not yet implemented")
		}

		return nil
	},
}

var newNotebookCmd = &cobra.Command{
	Use:   "notebook",
	Short: "Create a new notebook",
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := ".nocti/nocti.json"

		// Prompt for notebook name
		var notebookName string
		namePrompt := &survey.Input{
			Message: "Enter notebook name:",
		}
		err := survey.AskOne(namePrompt, &notebookName, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}
		notebookName = strings.TrimSpace(notebookName)

		// Read existing config
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		var config FullConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		// Create a map of existing IDs for quick lookup
		existingIDs := make(map[string]bool)
		for _, nb := range config.Notebooks {
			existingIDs[nb.ID] = true
		}

		// Generate a unique ID
		newID := generateID()
		for existingIDs[newID] {
			newID = generateID()
		}

		// Add new notebook
		newNB := Notebook{
			ID:        newID,
			Name:      notebookName,
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		config.Notebooks = append(config.Notebooks, newNB)

		// Save updated config
		updatedData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal updated config: %w", err)
		}

		if err := os.WriteFile(filename, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Successfully created notebook: %s\n", notebookName)
		return nil
	},
}

func init() {
	newCmd.AddCommand(newNotebookCmd)
	rootCmd.AddCommand(newCmd)
}
