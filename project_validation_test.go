package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSMFProjectFindsParentCheckout(t *testing.T) {
	ProjectDirectoryPath := t.TempDir()
	RequiredRelativePaths := []string{
		"playbooks/install-all.yml",
		"playbooks/install-docker.yml",
		"playbooks/build-smf-image.yml",
		"playbooks/install-smf.yml",
		"inventory/hosts.yml",
	}
	for _, RequiredRelativePath := range RequiredRelativePaths {
		RequiredPath := filepath.Join(ProjectDirectoryPath, RequiredRelativePath)
		if DirectoryError := os.MkdirAll(filepath.Dir(RequiredPath), 0755); DirectoryError != nil {
			t.Fatal(DirectoryError)
		}
		if WriteError := os.WriteFile(RequiredPath, []byte("---\n"), 0644); WriteError != nil {
			t.Fatal(WriteError)
		}
	}
	ChildDirectoryPath := filepath.Join(ProjectDirectoryPath, "nested", "directory")
	if DirectoryError := os.MkdirAll(ChildDirectoryPath, 0755); DirectoryError != nil {
		t.Fatal(DirectoryError)
	}
	Configuration := NewDefaultCLIConfiguration()
	Configuration.ProjectDirectoryPath = ChildDirectoryPath
	Project, ProjectError := ResolveSMFProject(Configuration)
	if ProjectError != nil {
		t.Fatal(ProjectError)
	}
	if Project.RootPath != ProjectDirectoryPath {
		t.Fatalf("project root = %q, want %q", Project.RootPath, ProjectDirectoryPath)
	}
}
