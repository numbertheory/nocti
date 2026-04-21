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

		// Verify directory creation
		if info, err := os.Stat("test-notebook"); os.IsNotExist(err) {
			t.Error("Expected directory 'test-notebook' to be created")
		} else if !info.IsDir() {
			t.Error("Expected 'test-notebook' to be a directory")
		}

		// Verify .nocti.json creation
		metadataPath := "test-notebook/.nocti.json"
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			t.Error("Expected '.nocti.json' to be created in notebook directory")
		} else {
			data, _ := os.ReadFile(metadataPath)
			var metadata map[string]string
			json.Unmarshal(data, &metadata)
			if metadata["id"] == "" || metadata["type"] != "notebook" || metadata["created_at"] == "" {
				t.Errorf("Metadata in .nocti.json is incorrect: %v", metadata)
			}
		}
	})

	t.Run("Create a notebook when directory already exists", func(t *testing.T) {
		dirName := "existing-notebook"
		if err := os.Mkdir(dirName, 0755); err != nil {
			t.Fatalf("Failed to create pre-existing directory: %v", err)
		}

		cmd.ResourceName = dirName
		cmd.Overwrite = false
		err := cmd.CreateResource("notebook")
		if err != nil {
			t.Errorf("CreateResource('notebook') failed when directory exists: %v", err)
		}

		// Verify directory still exists and .nocti.json is created
		if info, err := os.Stat(dirName); os.IsNotExist(err) {
			t.Error("Expected directory 'existing-notebook' to still exist")
		} else if !info.IsDir() {
			t.Error("Expected 'existing-notebook' to be a directory")
		}

		if _, err := os.Stat(dirName + "/.nocti.json"); os.IsNotExist(err) {
			t.Error("Expected '.nocti.json' to be created in existing notebook directory")
		}
	})

	t.Run("Fail when .nocti.json already exists and no overwrite flag", func(t *testing.T) {
		dirName := "no-overwrite"
		os.Mkdir(dirName, 0755)
		os.WriteFile(dirName+"/.nocti.json", []byte("{}"), 0644)

		cmd.ResourceName = dirName
		cmd.Overwrite = false
		err := cmd.CreateResource("notebook")
		if err == nil {
			t.Error("Expected error when .nocti.json exists and overwrite is false")
		}
	})

	t.Run("Succeed when .nocti.json already exists and overwrite flag is set", func(t *testing.T) {
		dirName := "yes-overwrite"
		os.Mkdir(dirName, 0755)
		os.WriteFile(dirName+"/.nocti.json", []byte("old content"), 0644)

		cmd.ResourceName = dirName
		cmd.Overwrite = true
		err := cmd.CreateResource("notebook")
		if err != nil {
			t.Errorf("Expected success when .nocti.json exists and overwrite is true: %v", err)
		}

		data, _ := os.ReadFile(dirName + "/.nocti.json")
		if string(data) == "old content" {
			t.Error("Expected .nocti.json to be overwritten")
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

		// Total notebooks should be 5 (1 from first test + 1 from second + 1 from successful overwrite test + 2 from this one)
		if len(config.Notebooks) != 5 {
			t.Errorf("Expected 5 notebooks total, got %d", len(config.Notebooks))
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
