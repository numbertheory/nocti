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
		// Register it first to satisfy the new uniqueness logic
		configData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(configData, &config)
		config.Notebooks = append(config.Notebooks, cmd.Notebook{ID: "exist1", Name: dirName, CreatedAt: "2026-04-21T00:00:00Z"})
		updatedConfig, _ := json.Marshal(config)
		os.WriteFile(".nocti/nocti.json", updatedConfig, 0644)

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
		// Register it first
		configData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(configData, &config)
		config.Notebooks = append(config.Notebooks, cmd.Notebook{ID: "noovr1", Name: dirName, CreatedAt: "2026-04-21T00:00:00Z"})
		updatedConfig, _ := json.Marshal(config)
		os.WriteFile(".nocti/nocti.json", updatedConfig, 0644)

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
		// Register it first
		configData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(configData, &config)
		config.Notebooks = append(config.Notebooks, cmd.Notebook{ID: "yesovr1", Name: dirName, CreatedAt: "2026-04-21T00:00:00Z"})
		updatedConfig, _ := json.Marshal(config)
		os.WriteFile(".nocti/nocti.json", updatedConfig, 0644)

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

		// Verify directory and .nocti.json
		if _, err := os.Stat("test-todo/.nocti.json"); os.IsNotExist(err) {
			t.Error("Expected 'test-todo/.nocti.json' to be created")
		}
	})

	t.Run("Create a new calendar", func(t *testing.T) {
		cmd.ResourceName = "test-calendar"
		err := cmd.CreateResource("calendar")
		if err != nil {
			t.Errorf("CreateResource('calendar') failed: %v", err)
		}

		// Verify file content
		updatedData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(updatedData, &config)

		if len(config.Calendars) != 1 {
			t.Errorf("Expected 1 calendar, got %d", len(config.Calendars))
		}
		if config.Calendars[0].Name != "test-calendar" {
			t.Errorf("Expected calendar name 'test-calendar', got '%s'", config.Calendars[0].Name)
		}

		// Verify directory and .nocti.json
		if _, err := os.Stat("test-calendar/.nocti.json"); os.IsNotExist(err) {
			t.Error("Expected 'test-calendar/.nocti.json' to be created")
		}
	})

	t.Run("Re-creating a notebook with same name should not duplicate config entry", func(t *testing.T) {
		cmd.ResourceName = "test-notebook" // already created in first test
		cmd.Overwrite = true
		err := cmd.CreateResource("notebook")
		if err != nil {
			t.Errorf("Expected success when re-creating notebook: %v", err)
		}

		updatedData, _ := os.ReadFile(".nocti/nocti.json")
		var config cmd.FullConfig
		json.Unmarshal(updatedData, &config)

		count := 0
		for _, nb := range config.Notebooks {
			if nb.Name == "test-notebook" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("Expected only 1 notebook entry for 'test-notebook', got %d", count)
		}
	})

	t.Run("Fail when folder exists but not in config", func(t *testing.T) {
		dirName := "unregistered-folder"
		os.Mkdir(dirName, 0755)

		cmd.ResourceName = dirName
		cmd.Overwrite = false
		err := cmd.CreateResource("notebook")
		if err == nil {
			t.Error("Expected error when folder exists but is not in config")
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

		// Total notebooks should be 8:
		// 1 (test-notebook)
		// 1 (existing-notebook - pre-registered)
		// 1 (no-overwrite - pre-registered)
		// 1 (yes-overwrite - pre-registered)
		// 1 (unregistered-folder - fails, so not added)
		// 1 (re-creation of test-notebook - not added)
		// 2 (nb-1, nb-2)
		// Wait, let's recount:
		// 1. "Create a new notebook" -> test-notebook (1)
		// 2. "Create a notebook when directory already exists" -> manually added "existing-notebook" (2)
		// 3. "Fail when .nocti.json already exists" -> manually added "no-overwrite" (3)
		// 4. "Succeed when .nocti.json already exists" -> manually added "yes-overwrite" (4)
		// 5. "Re-creating a notebook" -> test-notebook (already exists, count stays 4)
		// 6. "Fail when folder exists but not in config" -> unregistered-folder (fails, count stays 4)
		// 7. "Create multiple resources" -> nb-1, nb-2 (6 total)

		if len(config.Notebooks) != 6 {
			t.Errorf("Expected 6 notebooks total, got %d", len(config.Notebooks))
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
