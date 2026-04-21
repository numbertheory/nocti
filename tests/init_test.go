package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd(t *testing.T) {
	// Setup a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nocti-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change working directory to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	t.Run("Initialize project with flag", func(t *testing.T) {
		cmd.ProjectName = "test-project"
		err := cmd.InitCmd.RunE(cmd.InitCmd, []string{})
		if err != nil {
			t.Errorf("InitCmd.RunE failed: %v", err)
		}

		// Verify directory and file existence
		configPath := filepath.Join(".nocti", "nocti.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file %s was not created", configPath)
		}

		// Verify content
		data, _ := os.ReadFile(configPath)
		var config cmd.Config
		if err := json.Unmarshal(data, &config); err != nil {
			t.Errorf("Failed to unmarshal config: %v", err)
		}

		if config.Name != "test-project" {
			t.Errorf("Expected project name 'test-project', got '%s'", config.Name)
		}
	})

	t.Run("Initialize project when it already exists", func(t *testing.T) {
		// Try to init again in the same directory
		cmd.ProjectName = "another-project"
		err := cmd.InitCmd.RunE(cmd.InitCmd, []string{})
		if err == nil {
			t.Error("Expected error when initializing an already existing project, got nil")
		}
	})

	t.Run("Initialize project inside a resource folder", func(t *testing.T) {
		resourceDir, _ := os.MkdirTemp(tmpDir, "resource-dir-*")
		os.Chdir(resourceDir)
		defer os.Chdir(tmpDir)

		os.WriteFile(".nocti.json", []byte("{}"), 0644)

		cmd.ProjectName = "fail-project"
		err := cmd.InitCmd.RunE(cmd.InitCmd, []string{})
		if err == nil {
			t.Error("Expected error when initializing inside a resource folder, got nil")
		}
		if err != nil && err.Error() != "cannot run 'nocti init' inside a nocti resource directory" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})
}
