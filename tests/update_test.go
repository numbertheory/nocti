package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateCmd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-update-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// 1. Setup a nocti project
	os.Mkdir(".nocti", 0755)
	initialConfig := cmd.FullConfig{
		Name:    "test-project",
		Version: "1.0.0",
	}
	data, _ := json.Marshal(initialConfig)
	os.WriteFile(filepath.Join(".nocti", "nocti.json"), data, 0644)

	// 2. Create some resources manually (simulating existing folders with .nocti.json)
	resources := []struct {
		name string
		kind string
	}{
		{"nb1", "notebook"},
		{"todo1", "todo"},
		{"cal1", "calendar"},
	}

	for _, r := range resources {
		os.Mkdir(r.name, 0755)
		resMeta := map[string]string{
			"id":   r.name + "-id",
			"name": r.name,
			"type": r.kind,
		}
		resData, _ := json.Marshal(resMeta)
		os.WriteFile(filepath.Join(r.name, ".nocti.json"), resData, 0644)
	}

	// 3. Run update command
	err = cmd.UpdateCmd.RunE(cmd.UpdateCmd, []string{})
	if err != nil {
		t.Fatalf("UpdateCmd failed: %v", err)
	}

	// 4. Verify config was updated
	updatedData, _ := os.ReadFile(filepath.Join(".nocti", "nocti.json"))
	var config cmd.FullConfig
	json.Unmarshal(updatedData, &config)

	if len(config.Notebooks) != 1 || config.Notebooks[0].Name != "nb1" {
		t.Errorf("Expected 1 notebook 'nb1', got %v", config.Notebooks)
	}
	if len(config.Todos) != 1 || config.Todos[0].Name != "todo1" {
		t.Errorf("Expected 1 todo 'todo1', got %v", config.Todos)
	}
	if len(config.Calendars) != 1 || config.Calendars[0].Name != "cal1" {
		t.Errorf("Expected 1 calendar 'cal1', got %v", config.Calendars)
	}
}
