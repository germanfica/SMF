package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// AnsibleCoreVersion is deliberately pinned so the private control-node
// runtime is reproducible. Update it together with a tested project release.
const AnsibleCoreVersion = "2.21.4"

type AnsibleResolution struct {
	AnsiblePlaybookPath string
	RuntimePath         string
	UsesManagedRuntime  bool
}

func DescribeAnsibleResolution(Configuration CLIConfiguration) (AnsibleResolution, error) {
	if Configuration.AnsiblePlaybookPath != "" {
		RequestedPath, RequestedPathError := ResolveExecutablePath(Configuration.AnsiblePlaybookPath)
		if RequestedPathError != nil {
			return AnsibleResolution{}, RequestedPathError
		}
		return AnsibleResolution{AnsiblePlaybookPath: RequestedPath}, nil
	}
	if SystemAnsiblePlaybookPath, SystemAnsiblePlaybookError := exec.LookPath("ansible-playbook"); SystemAnsiblePlaybookError == nil {
		return AnsibleResolution{AnsiblePlaybookPath: SystemAnsiblePlaybookPath}, nil
	}
	RuntimePath, RuntimePathError := ResolveAnsibleRuntimePath(Configuration)
	if RuntimePathError != nil {
		return AnsibleResolution{}, RuntimePathError
	}
	ManagedAnsiblePlaybookPath := filepath.Join(RuntimePath, "bin", "ansible-playbook")
	if ExecutableFileExists(ManagedAnsiblePlaybookPath) {
		return AnsibleResolution{AnsiblePlaybookPath: ManagedAnsiblePlaybookPath, RuntimePath: RuntimePath, UsesManagedRuntime: true}, nil
	}
	return AnsibleResolution{RuntimePath: RuntimePath, UsesManagedRuntime: true}, nil
}

func EnsureAnsiblePlaybook(Configuration CLIConfiguration) (AnsibleResolution, error) {
	Resolution, ResolutionError := DescribeAnsibleResolution(Configuration)
	if ResolutionError != nil {
		return AnsibleResolution{}, ResolutionError
	}
	if Resolution.AnsiblePlaybookPath != "" {
		return Resolution, nil
	}
	if !Configuration.BootstrapAnsible {
		return AnsibleResolution{}, fmt.Errorf("ansible-playbook is unavailable and --no-bootstrap-ansible was selected")
	}
	if BootstrapError := BootstrapManagedAnsibleRuntime(Resolution.RuntimePath); BootstrapError != nil {
		return AnsibleResolution{}, BootstrapError
	}
	ManagedAnsiblePlaybookPath := filepath.Join(Resolution.RuntimePath, "bin", "ansible-playbook")
	if !ExecutableFileExists(ManagedAnsiblePlaybookPath) {
		return AnsibleResolution{}, fmt.Errorf("ansible-core installation did not create %q", ManagedAnsiblePlaybookPath)
	}
	Resolution.AnsiblePlaybookPath = ManagedAnsiblePlaybookPath
	return Resolution, nil
}

func BootstrapManagedAnsibleRuntime(RuntimePath string) error {
	PythonPath, PythonPathError := exec.LookPath("python3")
	if PythonPathError != nil {
		return fmt.Errorf("find python3 required to bootstrap ansible-core: %w", PythonPathError)
	}
	if !PythonProvidesVenv(PythonPath) {
		if PythonVenvPackageError := InstallPythonVenvPackage(); PythonVenvPackageError != nil {
			return PythonVenvPackageError
		}
		if !PythonProvidesVenv(PythonPath) {
			return fmt.Errorf("python3 still cannot import venv after installing python3-venv")
		}
	}
	if RuntimeInformation, RuntimeInformationError := os.Stat(RuntimePath); RuntimeInformationError == nil {
		if !RuntimeInformation.IsDir() {
			return fmt.Errorf("Ansible runtime path %q is not a directory", RuntimePath)
		}
		if !ExecutableFileExists(filepath.Join(RuntimePath, "bin", "python")) {
			return fmt.Errorf("refusing to overwrite incomplete Ansible runtime %q", RuntimePath)
		}
	} else if !os.IsNotExist(RuntimeInformationError) {
		return fmt.Errorf("inspect Ansible runtime %q: %w", RuntimePath, RuntimeInformationError)
	}
	if ParentDirectoryError := os.MkdirAll(filepath.Dir(RuntimePath), 0755); ParentDirectoryError != nil {
		return fmt.Errorf("create Ansible runtime parent directory: %w", ParentDirectoryError)
	}
	if !ExecutableFileExists(filepath.Join(RuntimePath, "bin", "python")) {
		VenvOperation := CommandOperation{
			Name:           "Create private ansible-core environment",
			ExecutablePath: PythonPath,
			Arguments:      []string{"-m", "venv", RuntimePath},
		}
		if VenvError := ExecuteCommandOperation(VenvOperation); VenvError != nil {
			return VenvError
		}
	}
	RuntimePythonPath := filepath.Join(RuntimePath, "bin", "python")
	UpgradePipOperation := CommandOperation{
		Name:           "Upgrade private environment pip",
		ExecutablePath: RuntimePythonPath,
		Arguments:      []string{"-m", "pip", "install", "--disable-pip-version-check", "--upgrade", "pip"},
	}
	if UpgradePipError := ExecuteCommandOperation(UpgradePipOperation); UpgradePipError != nil {
		return UpgradePipError
	}
	InstallAnsibleOperation := CommandOperation{
		Name:           "Install pinned ansible-core",
		ExecutablePath: RuntimePythonPath,
		Arguments:      []string{"-m", "pip", "install", "--disable-pip-version-check", "ansible-core==" + AnsibleCoreVersion},
	}
	if InstallAnsibleError := ExecuteCommandOperation(InstallAnsibleOperation); InstallAnsibleError != nil {
		return InstallAnsibleError
	}
	return nil
}

func PythonProvidesVenv(PythonPath string) bool {
	ValidationCommand := exec.Command(PythonPath, "-c", "import venv")
	return ValidationCommand.Run() == nil
}

func InstallPythonVenvPackage() error {
	if _, APTPathError := exec.LookPath("apt-get"); APTPathError != nil {
		return fmt.Errorf("python3 lacks venv and apt-get is unavailable; install the Python venv package manually")
	}
	if _, SudoPathError := exec.LookPath("sudo"); SudoPathError != nil {
		return fmt.Errorf("python3 lacks venv and sudo is unavailable; install the Python venv package manually")
	}
	UpdateOperation := CommandOperation{
		Name:           "Update package metadata for Python venv",
		ExecutablePath: "sudo",
		Arguments:      []string{"apt-get", "update"},
	}
	if UpdateError := ExecuteCommandOperation(UpdateOperation); UpdateError != nil {
		return UpdateError
	}
	InstallOperation := CommandOperation{
		Name:           "Install Python venv support",
		ExecutablePath: "sudo",
		Arguments:      []string{"apt-get", "install", "--yes", "python3-venv"},
	}
	return ExecuteCommandOperation(InstallOperation)
}

func BuildAnsiblePlaybookOperation(Configuration CLIConfiguration, Project SMFProject, Resolution AnsibleResolution, AskVaultPassword bool) (CommandOperation, error) {
	ExtraVariables := map[string]interface{}{
		"SMF_DOCKER_COMMAND_BECOME": true,
		"SMF_INSTALLER_ENABLED": Configuration.InstallerEnabled,
		"SMF_PROJECT_PATH":     Project.RootPath,
		"SMF_PUBLISHED_PORT":   "",
	}
	if Configuration.ExposureMode == ExposureModePublishedPort {
		ExtraVariables["SMF_PUBLISHED_PORT"] = fmt.Sprintf("%d", Configuration.PublishedPort)
	}
	return BuildSMFAnsiblePlaybookOperation(
		Configuration,
		Project,
		Resolution,
		AskVaultPassword,
		ExtraVariables,
		Project.InstallPlaybookPath,
		"Install SMF through Ansible",
	)
}

func BuildConfigureAnsiblePlaybookOperation(Configuration CLIConfiguration, Project SMFProject, Resolution AnsibleResolution, AskVaultPassword bool) (CommandOperation, error) {
	ExtraVariables := map[string]interface{}{
		"SMF_PUBLISHED_PORT": "",
		"SMF_FORUM_URL":      Configuration.ForumURL,
	}
	if Configuration.ExposureMode == ExposureModePublishedPort {
		ExtraVariables["SMF_PUBLISHED_PORT"] = fmt.Sprintf("%d", Configuration.PublishedPort)
	}
	return BuildSMFAnsiblePlaybookOperation(
		Configuration,
		Project,
		Resolution,
		AskVaultPassword,
		ExtraVariables,
		Project.ConfigurePlaybookPath,
		"Configure deployed SMF through Ansible",
	)
}

func BuildSMFAnsiblePlaybookOperation(Configuration CLIConfiguration, Project SMFProject, Resolution AnsibleResolution, AskVaultPassword bool, ExtraVariables map[string]interface{}, PlaybookPath string, OperationName string) (CommandOperation, error) {
	if Configuration.TargetHosts != "" {
		ExtraVariables["TARGET_HOSTS"] = Configuration.TargetHosts
	}
	ExtraVariablesJSON, ExtraVariablesJSONError := json.Marshal(ExtraVariables)
	if ExtraVariablesJSONError != nil {
		return CommandOperation{}, fmt.Errorf("encode Ansible extra variables: %w", ExtraVariablesJSONError)
	}
	Arguments := []string{"--inventory", Project.InventoryPath}
	if ShouldAskBecomePassword(Configuration) {
		Arguments = append(Arguments, "--ask-become-pass")
	}
	if Configuration.VaultPasswordFilePath != "" {
		AbsoluteVaultPasswordFilePath, VaultPasswordFilePathError := filepath.Abs(Configuration.VaultPasswordFilePath)
		if VaultPasswordFilePathError != nil {
			return CommandOperation{}, fmt.Errorf("resolve Vault password file: %w", VaultPasswordFilePathError)
		}
		Arguments = append(Arguments, "--vault-password-file", AbsoluteVaultPasswordFilePath)
	} else if AskVaultPassword {
		Arguments = append(Arguments, "--ask-vault-pass")
	}
	Arguments = append(Arguments, "--extra-vars", string(ExtraVariablesJSON), PlaybookPath)
	return CommandOperation{
		Name:             OperationName,
		ExecutablePath:   Resolution.AnsiblePlaybookPath,
		Arguments:        Arguments,
		WorkingDirectory: Project.RootPath,
	}, nil
}

func ResolveExecutablePath(RequestedPath string) (string, error) {
	if filepath.IsAbs(RequestedPath) || stringsContainsPathSeparator(RequestedPath) {
		AbsolutePath, AbsolutePathError := filepath.Abs(RequestedPath)
		if AbsolutePathError != nil {
			return "", fmt.Errorf("resolve executable path: %w", AbsolutePathError)
		}
		if !ExecutableFileExists(AbsolutePath) {
			return "", fmt.Errorf("requested executable %q is unavailable or not executable", AbsolutePath)
		}
		return AbsolutePath, nil
	}
	ExecutablePath, ExecutablePathError := exec.LookPath(RequestedPath)
	if ExecutablePathError != nil {
		return "", fmt.Errorf("find requested executable %q: %w", RequestedPath, ExecutablePathError)
	}
	return ExecutablePath, nil
}

func stringsContainsPathSeparator(Value string) bool {
	return filepath.Base(Value) != Value
}

func ExecutableFileExists(Path string) bool {
	FileInformation, FileInformationError := os.Stat(Path)
	return FileInformationError == nil && FileInformation.Mode().IsRegular() && FileInformation.Mode()&0111 != 0
}
