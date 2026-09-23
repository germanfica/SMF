package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

type CLICommand string

const (
	CLICommandHelp    CLICommand = "help"
	CLICommandVersion CLICommand = "version"
	CLICommandInstall CLICommand = "install"
	CLICommandList    CLICommand = "list"
)

type ExposureMode string

const (
	ExposureModeNetworkOnly  ExposureMode = "network-only"
	ExposureModePublishedPort ExposureMode = "published-port"
)

type PromptMode string

const (
	PromptModeAutomatic PromptMode = "automatic"
	PromptModeEnabled   PromptMode = "enabled"
	PromptModeDisabled  PromptMode = "disabled"
)

type CLIConfiguration struct {
	Command                  CLICommand
	ApplyChanges             bool
	ProjectDirectoryPath     string
	InventoryPath            string
	TargetHosts              string
	ExposureMode             ExposureMode
	PublishedPort            int
	ExposureWasSpecified     bool
	Interactive              bool
	InstallerEnabled         bool
	AskBecomePassword        PromptMode
	AskVaultPassword         PromptMode
	VaultPasswordFilePath    string
	AnsiblePlaybookPath      string
	AnsibleRuntimePath       string
	BootstrapAnsible         bool
	DeploymentDirectoryPath  string
}

func NewDefaultCLIConfiguration() CLIConfiguration {
	return CLIConfiguration{
		ExposureMode:            ExposureModeNetworkOnly,
		InstallerEnabled:        true,
		AskBecomePassword:       PromptModeAutomatic,
		AskVaultPassword:        PromptModeAutomatic,
		BootstrapAnsible:        true,
		DeploymentDirectoryPath: "/opt/smf",
	}
}

func ParseCLIConfiguration(Arguments []string) (CLIConfiguration, error) {
	if len(Arguments) == 0 {
		Configuration := NewDefaultCLIConfiguration()
		Configuration.Command = CLICommandHelp
		return Configuration, nil
	}

	switch Arguments[0] {
	case "--help", "-h", "help":
		if len(Arguments) != 1 {
			return CLIConfiguration{}, fmt.Errorf("%s does not accept arguments", Arguments[0])
		}
		Configuration := NewDefaultCLIConfiguration()
		Configuration.Command = CLICommandHelp
		return Configuration, nil
	case "--version", "version":
		if len(Arguments) != 1 {
			return CLIConfiguration{}, fmt.Errorf("%s does not accept arguments", Arguments[0])
		}
		Configuration := NewDefaultCLIConfiguration()
		Configuration.Command = CLICommandVersion
		return Configuration, nil
	case "--install":
		return ParseInstallConfiguration(Arguments[1:], true)
	case "install":
		return ParseInstallConfiguration(Arguments[1:], false)
	case "--list", "list":
		return ParseListConfiguration(Arguments[1:])
	default:
		return CLIConfiguration{}, fmt.Errorf("unknown command or option %q", Arguments[0])
	}
}

func ParseInstallConfiguration(Arguments []string, ApplyByDefault bool) (CLIConfiguration, error) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.Command = CLICommandInstall
	Configuration.ApplyChanges = ApplyByDefault

	Flags := flag.NewFlagSet("smf install", flag.ContinueOnError)
	Flags.SetOutput(io.Discard)
	ProjectDirectoryPath := Flags.String("project-dir", "", "SMF repository checkout directory")
	InventoryPath := Flags.String("inventory", "", "Ansible inventory path, relative to the project when not absolute")
	TargetHosts := Flags.String("target", "", "Ansible target hosts expression")
	PublishedPort := Flags.Int("port", 0, "publish this host TCP port to SMF container port 80")
	NetworkOnly := Flags.Bool("network-only", false, "keep SMF reachable only on its Docker networks")
	ApplyChanges := Flags.Bool("apply", ApplyByDefault, "apply the installation plan")
	NonInteractive := Flags.Bool("non-interactive", false, "do not prompt for exposure or confirmation")
	DisableInstaller := Flags.Bool("disable-installer", false, "disable SMF's web installer during deployment")
	AskBecomePassword := Flags.Bool("ask-become-pass", false, "ask Ansible for the sudo password")
	NoAskBecomePassword := Flags.Bool("no-ask-become-pass", false, "do not ask Ansible for the sudo password")
	AskVaultPassword := Flags.Bool("ask-vault-pass", false, "ask Ansible for the Vault password")
	NoAskVaultPassword := Flags.Bool("no-ask-vault-pass", false, "do not ask Ansible for the Vault password")
	VaultPasswordFilePath := Flags.String("vault-password-file", "", "path to an Ansible Vault password file")
	AnsiblePlaybookPath := Flags.String("ansible-playbook", "", "path to ansible-playbook")
	AnsibleRuntimePath := Flags.String("ansible-runtime", "", "private ansible-core virtual environment path")
	NoBootstrapAnsible := Flags.Bool("no-bootstrap-ansible", false, "fail instead of creating the private ansible-core environment")

	if FlagError := Flags.Parse(Arguments); FlagError != nil {
		return CLIConfiguration{}, FlagError
	}
	if len(Flags.Args()) != 0 {
		return CLIConfiguration{}, fmt.Errorf("unexpected argument %q", Flags.Args()[0])
	}

	VisitedFlags := make(map[string]bool)
	Flags.Visit(func(FlagValue *flag.Flag) {
		VisitedFlags[FlagValue.Name] = true
	})
	if *NetworkOnly && VisitedFlags["port"] {
		return CLIConfiguration{}, fmt.Errorf("--network-only and --port cannot be used together")
	}
	if VisitedFlags["port"] && (*PublishedPort < 1 || *PublishedPort > 65535) {
		return CLIConfiguration{}, fmt.Errorf("--port must be between 1 and 65535")
	}
	if *AskBecomePassword && *NoAskBecomePassword {
		return CLIConfiguration{}, fmt.Errorf("--ask-become-pass and --no-ask-become-pass cannot be used together")
	}
	if *AskVaultPassword && *NoAskVaultPassword {
		return CLIConfiguration{}, fmt.Errorf("--ask-vault-pass and --no-ask-vault-pass cannot be used together")
	}
	if *VaultPasswordFilePath != "" && *AskVaultPassword {
		return CLIConfiguration{}, fmt.Errorf("--vault-password-file and --ask-vault-pass cannot be used together")
	}

	Configuration.ApplyChanges = *ApplyChanges
	Configuration.ProjectDirectoryPath = *ProjectDirectoryPath
	Configuration.InventoryPath = *InventoryPath
	Configuration.TargetHosts = *TargetHosts
	Configuration.PublishedPort = *PublishedPort
	Configuration.ExposureWasSpecified = VisitedFlags["port"] || *NetworkOnly
	Configuration.Interactive = !*NonInteractive && StandardInputIsTerminal()
	Configuration.InstallerEnabled = !*DisableInstaller
	Configuration.VaultPasswordFilePath = *VaultPasswordFilePath
	Configuration.AnsiblePlaybookPath = *AnsiblePlaybookPath
	Configuration.AnsibleRuntimePath = *AnsibleRuntimePath
	Configuration.BootstrapAnsible = !*NoBootstrapAnsible
	if *PublishedPort != 0 {
		Configuration.ExposureMode = ExposureModePublishedPort
	}
	if *AskBecomePassword {
		Configuration.AskBecomePassword = PromptModeEnabled
	}
	if *NoAskBecomePassword {
		Configuration.AskBecomePassword = PromptModeDisabled
	}
	if *AskVaultPassword {
		Configuration.AskVaultPassword = PromptModeEnabled
	}
	if *NoAskVaultPassword {
		Configuration.AskVaultPassword = PromptModeDisabled
	}

	// --install is the documented compatibility shortcut. It must be useful in
	// scripts, so an omitted exposure means Docker-network-only rather than a prompt.
	if ApplyByDefault && !Configuration.ExposureWasSpecified {
		Configuration.ExposureWasSpecified = true
	}
	return Configuration, nil
}

func ParseListConfiguration(Arguments []string) (CLIConfiguration, error) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.Command = CLICommandList
	Flags := flag.NewFlagSet("smf list", flag.ContinueOnError)
	Flags.SetOutput(io.Discard)
	DeploymentDirectoryPath := Flags.String("deployment-dir", Configuration.DeploymentDirectoryPath, "local SMF Compose deployment directory")
	if FlagError := Flags.Parse(Arguments); FlagError != nil {
		return CLIConfiguration{}, FlagError
	}
	if len(Flags.Args()) != 0 {
		return CLIConfiguration{}, fmt.Errorf("unexpected argument %q", Flags.Args()[0])
	}
	AbsoluteDeploymentDirectoryPath, PathError := filepath.Abs(*DeploymentDirectoryPath)
	if PathError != nil {
		return CLIConfiguration{}, fmt.Errorf("resolve deployment directory: %w", PathError)
	}
	Configuration.DeploymentDirectoryPath = AbsoluteDeploymentDirectoryPath
	return Configuration, nil
}

func PrintUsage(Writer io.Writer) {
	fmt.Fprint(Writer, `Usage:
  smf install [options]
  smf list [options]
  smf --install [options]
  smf --version
  smf --help

Commands:
  install  Plan an SMF installation and, after confirmation or --apply, run
           playbooks/install-all.yml.
  list     Show the local SMF Compose service state.

Install options:
  --port PORT              Publish PORT on the host as PORT:80.
  --network-only           Keep SMF on Docker networks only (the default).
  --apply                  Apply a non-interactive installation plan.
  --disable-installer      Disable SMF's web installer during this deployment.
  --project-dir PATH       Repository containing the SMF playbooks.
  --inventory PATH         Inventory file, relative to the project by default.
  --target HOSTS           Override TARGET_HOSTS for the playbooks.
  --ask-become-pass        Always ask Ansible for the sudo password.
  --no-ask-become-pass     Never ask Ansible for the sudo password.
  --ask-vault-pass         Always ask for an Ansible Vault password.
  --no-ask-vault-pass      Never ask for an Ansible Vault password.
  --vault-password-file P  Use an Ansible Vault password file.
  --ansible-playbook PATH  Use this ansible-playbook executable.
  --ansible-runtime PATH   Store the managed private Ansible environment here.
  --no-bootstrap-ansible   Do not install ansible-core if it is unavailable.
  --non-interactive        Do not ask exposure or confirmation questions.

Compatibility:
  smf --install is equivalent to smf install --apply. With no exposure option,
  it uses Docker-network-only exposure so it is safe for scripts.
`)
}
