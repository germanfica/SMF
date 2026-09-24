package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SMFProject struct {
	RootPath              string
	InventoryPath         string
	InstallPlaybookPath   string
	ConfigurePlaybookPath string
	VaultFilePath         string
}

func ResolveSMFProject(Configuration CLIConfiguration) (SMFProject, error) {
	ProjectRootPath, ProjectRootError := ResolveSMFProjectRoot(Configuration.ProjectDirectoryPath)
	if ProjectRootError != nil {
		return SMFProject{}, ProjectRootError
	}
	RequiredRelativePaths := []string{
		"playbooks/install-all.yml",
		"playbooks/install-docker.yml",
		"playbooks/build-smf-image.yml",
		"playbooks/install-smf.yml",
		"playbooks/configure-smf.yml",
		"scripts/configure-smf-url-settings.php",
		"inventory/hosts.yml",
	}
	for _, RequiredRelativePath := range RequiredRelativePaths {
		RequiredPath := filepath.Join(ProjectRootPath, RequiredRelativePath)
		if RequiredFileError := ValidateRegularFile(RequiredPath); RequiredFileError != nil {
			return SMFProject{}, RequiredFileError
		}
	}
	InventoryPath, InventoryPathError := ResolveInventoryPath(ProjectRootPath, Configuration.InventoryPath)
	if InventoryPathError != nil {
		return SMFProject{}, InventoryPathError
	}
	return SMFProject{
		RootPath:              ProjectRootPath,
		InventoryPath:         InventoryPath,
		InstallPlaybookPath:   filepath.Join(ProjectRootPath, "playbooks", "install-all.yml"),
		ConfigurePlaybookPath: filepath.Join(ProjectRootPath, "playbooks", "configure-smf.yml"),
		VaultFilePath:         filepath.Join(ProjectRootPath, "inventory", "group_vars", "smf", "vault.yml"),
	}, nil
}

func ResolveSMFProjectRoot(SelectedProjectDirectoryPath string) (string, error) {
	StartingDirectoryPath := SelectedProjectDirectoryPath
	if StartingDirectoryPath == "" {
		CurrentWorkingDirectoryPath, CurrentWorkingDirectoryError := os.Getwd()
		if CurrentWorkingDirectoryError != nil {
			return "", fmt.Errorf("get current working directory: %w", CurrentWorkingDirectoryError)
		}
		StartingDirectoryPath = CurrentWorkingDirectoryPath
	}
	AbsoluteStartingDirectoryPath, StartingDirectoryPathError := filepath.Abs(StartingDirectoryPath)
	if StartingDirectoryPathError != nil {
		return "", fmt.Errorf("resolve project directory: %w", StartingDirectoryPathError)
	}
	DirectoryPath := AbsoluteStartingDirectoryPath
	for {
		InstallPlaybookPath := filepath.Join(DirectoryPath, "playbooks", "install-all.yml")
		if FileInformation, FileError := os.Stat(InstallPlaybookPath); FileError == nil && FileInformation.Mode().IsRegular() {
			return DirectoryPath, nil
		}
		ParentDirectoryPath := filepath.Dir(DirectoryPath)
		if ParentDirectoryPath == DirectoryPath {
			break
		}
		DirectoryPath = ParentDirectoryPath
	}
	return "", fmt.Errorf("could not find playbooks/install-all.yml; run smf from the SMF checkout or pass --project-dir")
}

func ResolveInventoryPath(ProjectRootPath string, SelectedInventoryPath string) (string, error) {
	InventoryPath := SelectedInventoryPath
	if InventoryPath == "" {
		InventoryPath = filepath.Join(ProjectRootPath, "inventory", "hosts.yml")
	} else if !filepath.IsAbs(InventoryPath) {
		InventoryPath = filepath.Join(ProjectRootPath, InventoryPath)
	}
	AbsoluteInventoryPath, InventoryPathError := filepath.Abs(InventoryPath)
	if InventoryPathError != nil {
		return "", fmt.Errorf("resolve inventory path: %w", InventoryPathError)
	}
	if ValidationError := ValidateRegularFile(AbsoluteInventoryPath); ValidationError != nil {
		return "", ValidationError
	}
	return AbsoluteInventoryPath, nil
}

func ValidateRegularFile(Path string) error {
	FileInformation, FileError := os.Stat(Path)
	if FileError != nil {
		return fmt.Errorf("inspect required file %q: %w", Path, FileError)
	}
	if !FileInformation.Mode().IsRegular() {
		return fmt.Errorf("required path %q is not a regular file", Path)
	}
	return nil
}

func ResolveAnsibleRuntimePath(Configuration CLIConfiguration) (string, error) {
	RuntimePath := Configuration.AnsibleRuntimePath
	if RuntimePath == "" {
		StateHomePath := os.Getenv("XDG_STATE_HOME")
		if StateHomePath == "" {
			UserHomeDirectoryPath, UserHomeDirectoryError := os.UserHomeDir()
			if UserHomeDirectoryError != nil {
				return "", fmt.Errorf("resolve user home directory: %w", UserHomeDirectoryError)
			}
			StateHomePath = filepath.Join(UserHomeDirectoryPath, ".local", "state")
		}
		RuntimePath = filepath.Join(StateHomePath, "smf", "ansible-core-"+AnsibleCoreVersion)
	}
	AbsoluteRuntimePath, RuntimePathError := filepath.Abs(RuntimePath)
	if RuntimePathError != nil {
		return "", fmt.Errorf("resolve Ansible runtime path: %w", RuntimePathError)
	}
	return AbsoluteRuntimePath, nil
}

func ShouldAskVaultPassword(Configuration CLIConfiguration, Project SMFProject) (bool, error) {
	if Configuration.VaultPasswordFilePath != "" {
		AbsoluteVaultPasswordFilePath, VaultPasswordFilePathError := filepath.Abs(Configuration.VaultPasswordFilePath)
		if VaultPasswordFilePathError != nil {
			return false, fmt.Errorf("resolve Vault password file: %w", VaultPasswordFilePathError)
		}
		if ValidationError := ValidateRegularFile(AbsoluteVaultPasswordFilePath); ValidationError != nil {
			return false, ValidationError
		}
		return false, nil
	}
	switch Configuration.AskVaultPassword {
	case PromptModeEnabled:
		return true, nil
	case PromptModeDisabled:
		return false, nil
	}
	VaultFileContents, VaultFileError := os.ReadFile(Project.VaultFilePath)
	if os.IsNotExist(VaultFileError) {
		return false, nil
	}
	if VaultFileError != nil {
		return false, fmt.Errorf("read Vault file %q: %w", Project.VaultFilePath, VaultFileError)
	}
	return strings.HasPrefix(string(VaultFileContents), "$ANSIBLE_VAULT;"), nil
}

func ShouldAskBecomePassword(Configuration CLIConfiguration) bool {
	switch Configuration.AskBecomePassword {
	case PromptModeEnabled:
		return true
	case PromptModeDisabled:
		return false
	default:
		return os.Geteuid() != 0
	}
}
