package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func RunSMFList(Configuration CLIConfiguration) error {
	ComposeFilePath := filepath.Join(Configuration.DeploymentDirectoryPath, "docker-compose.yml")
	if ValidationError := ValidateRegularFile(ComposeFilePath); ValidationError != nil {
		return fmt.Errorf("SMF deployment is unavailable: %w", ValidationError)
	}
	Arguments := []string{"compose", "--file", ComposeFilePath}
	PortOverridePath := filepath.Join(Configuration.DeploymentDirectoryPath, "docker-compose.smf.override.yml")
	if ExecutableFileExists(PortOverridePath) || RegularFileExists(PortOverridePath) {
		Arguments = append(Arguments, "--file", PortOverridePath)
	}
	Arguments = append(Arguments, "ps")
	Operation := CommandOperation{
		Name:             "List SMF Compose services",
		ExecutablePath:   "docker",
		Arguments:        Arguments,
		WorkingDirectory: Configuration.DeploymentDirectoryPath,
	}
	fmt.Fprintln(os.Stdout, "SMF service state:")
	return ExecuteCommandOperation(Operation)
}

func RegularFileExists(Path string) bool {
	FileInformation, FileInformationError := os.Stat(Path)
	return FileInformationError == nil && FileInformation.Mode().IsRegular()
}
