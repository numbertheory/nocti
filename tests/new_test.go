package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"testing"
)

func TestCreateResource(t *testing.T) {
	// Setup a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nocti-resource-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change working directory to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create .nocti directory and initial nocti.json
	if err := os.Mkdir(".nocti", 0755); err != nil {
		t.Fatalf("Failed to create .nocti directory: %v", err)
	}

	initialConfig := cmd.FullConfig{
		Name:    "test-project",
		Version: "test-version",
	}
	data, _ := json.Marshal(initialConfig)
	if err := os.WriteFile(".nocti/nocti.json", data, 0644); err != nil {
		t.Fatalf("Failed to write initial nocti.json: %v", err)
	}

	t.Run("Create a new notebook", func(t *testing.T) {
		cmd.ResourceName = "test-notebook"
		err := cmd.CreateResource("notebook")
		if err != nil {
			t.Errorf("CreateResource('notebook') failed: %v", err)
		}

		// Verify file content
		updatedData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(updatedData, &config)

		if len(config.Notebooks) != 1 {
			t.Errorf("Expected 1 notebook, got %d", len(config.Notebooks))
		}
		if config.Notebooks[0].Name != "test-notebook" {
			t.Errorf("Expected notebook name 'test-notebook', got '%s'", config.Notebooks[0].Name)
		}
		if config.Notebooks[0].ID == "" {
			t.Error("Notebook ID should not be empty")
		}
	})

	t.Run("Create a new todo list", func(t *testing.T) {
		cmd.ResourceName = "test-todo"
		err := cmd.CreateResource("todo")
		if err != nil {
			t.Errorf("CreateResource('todo') failed: %v", err)
		}

		// Verify file content
		updatedData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(updatedData, &config)

		if len(config.Todos) != 1 {
			t.Errorf("Expected 1 todo list, got %d", len(config.Todos))
		}
		if config.Todos[0].Name != "test-todo" {
			t.Errorf("Expected todo name 'test-todo', got '%s'", config.Todos[0].Name)
		}
	})

	t.Run("Create multiple resources and check unique IDs", func(t *testing.T) {
		cmd.ResourceName = "nb-1"
		cmd.CreateResource("notebook")
		cmd.ResourceName = "nb-2"
		cmd.CreateResource("notebook")

		updatedData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(updatedData, &config)

		// Total notebooks should be 3 (1 from first test + 2 from this one)
		if len(config.Notebooks) != 3 {
			t.Errorf("Expected 3 notebooks total, got %d", len(config.Notebooks))
		}

		ids := make(map[string]bool)
		for _, nb := range config.Notebooks {
			if ids[nb.ID] {
				t.Errorf("Duplicate ID found: %s", nb.ID)
			}
			ids[nb.ID] = true
		}
	})
}

func TestGenerateID(t *testing.T) {
	id := cmd.GenerateID()
	if len(id) != 6 {
		t.Errorf("Expected ID length of 6, got %d", len(id))
	}
}
