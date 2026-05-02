package tests

import (
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateID_Internal(t *testing.T) {
	id1 := cmd.GenerateID()
	id2 := cmd.GenerateID()

	if len(id1) != 6 {
		t.Errorf("Expected ID length 6, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("Generated IDs should be random, but got two identical ones")
	}
}

func TestFindProjectRoot_Internal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-root-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	os.MkdirAll(filepath.Join(tmpDir, ".nocti"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".nocti", "nocti.json"), []byte("{}"), 0644)

	childDir := filepath.Join(tmpDir, "a", "b")
	os.MkdirAll(childDir, 0755)

	os.Chdir(childDir)

	root, err := cmd.FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	if filepath.Base(root) != filepath.Base(tmpDir) {
		t.Errorf("Expected root %s, got %s", tmpDir, root)
	}

	os.Chdir(os.TempDir())
	_, err = cmd.FindProjectRoot()
	if err == nil {
		t.Error("Expected error when no project root found, got nil")
	}
}
