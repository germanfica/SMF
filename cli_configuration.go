package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
)

type CLICommand string

const (
	CLICommandHelp      CLICommand = "help"
	CLICommandVersion   CLICommand = "version"
	CLICommandInstall   CLICommand = "install"
	CLICommandConfigure CLICommand = "configure"
	CLICommandList      CLICommand = "list"
)

type ExposureMode string

const (
	ExposureModeNetworkOnly   ExposureMode = "network-only"
	ExposureModePublishedPort ExposureMode = "published-port"
)

type PromptMode string

const (
	PromptModeAutomatic PromptMode = "automatic"
	PromptModeEnabled   PromptMode = "enabled"
	PromptModeDisabled  PromptMode = "disabled"
)

type CLIConfiguration struct {
	Command                 CLICommand
	ApplyChanges            bool
	ProjectDirectoryPath    string
	InventoryPath           string
	TargetHosts             string
	ExposureMode            ExposureMode
	PublishedPort           int
	ExposureWasSpecified    bool
	ForumURL                string
	ForumURLWasSpecified    bool
	Interactive             bool
	InstallerEnabled        bool
	AskBecomePassword       PromptMode
	AskVaultPassword        PromptMode
	VaultPasswordFilePath   string
	AnsiblePlaybookPath     string
	AnsibleRuntimePath      string
	BootstrapAnsible        bool
	DeploymentDirectoryPath string
}

type OperationFlags struct {
	ProjectDirectoryPath  *string
	InventoryPath         *string
	TargetHosts           *string
	ApplyChanges          *bool
	NonInteractive        *bool
	AskBecomePassword     *bool
	NoAskBecomePassword   *bool
	AskVaultPassword      *bool
	NoAskVaultPassword    *bool
	VaultPasswordFilePath *string
	AnsiblePlaybookPath   *string
	AnsibleRuntimePath    *string
	NoBootstrapAnsible    *bool
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
	case "install":
		return ParseInstallConfiguration(Arguments[1:])
	case "configure":
		return ParseConfigureConfiguration(Arguments[1:])
	case "--list", "list":
		return ParseListConfiguration(Arguments[1:])
	default:
		return CLIConfiguration{}, fmt.Errorf("unknown command or option %q", Arguments[0])
	}
}

func RegisterOperationFlags(Flags *flag.FlagSet) OperationFlags {
	return OperationFlags{
		ProjectDirectoryPath:  Flags.String("project-dir", "", "SMF repository checkout directory"),
		InventoryPath:         Flags.String("inventory", "", "Ansible inventory path, relative to the project when not absolute"),
		TargetHosts:           Flags.String("target", "", "Ansible target hosts expression"),
		ApplyChanges:          Flags.Bool("apply", false, "apply the plan"),
		NonInteractive:        Flags.Bool("non-interactive", false, "do not prompt for settings or confirmation"),
		AskBecomePassword:     Flags.Bool("ask-become-pass", false, "ask Ansible for the sudo password"),
		NoAskBecomePassword:   Flags.Bool("no-ask-become-pass", false, "do not ask Ansible for the sudo password"),
		AskVaultPassword:      Flags.Bool("ask-vault-pass", false, "ask Ansible for the Vault password"),
		NoAskVaultPassword:    Flags.Bool("no-ask-vault-pass", false, "do not ask Ansible for the Vault password"),
		VaultPasswordFilePath: Flags.String("vault-password-file", "", "path to an Ansible Vault password file"),
		AnsiblePlaybookPath:   Flags.String("ansible-playbook", "", "path to ansible-playbook"),
		AnsibleRuntimePath:    Flags.String("ansible-runtime", "", "private ansible-core virtual environment path"),
		NoBootstrapAnsible:    Flags.Bool("no-bootstrap-ansible", false, "fail instead of creating the private ansible-core environment"),
	}
}

func ApplyOperationFlags(Configuration *CLIConfiguration, Values OperationFlags) error {
	if *Values.AskBecomePassword && *Values.NoAskBecomePassword {
		return fmt.Errorf("--ask-become-pass and --no-ask-become-pass cannot be used together")
	}
	if *Values.AskVaultPassword && *Values.NoAskVaultPassword {
		return fmt.Errorf("--ask-vault-pass and --no-ask-vault-pass cannot be used together")
	}
	if *Values.VaultPasswordFilePath != "" && *Values.AskVaultPassword {
		return fmt.Errorf("--vault-password-file and --ask-vault-pass cannot be used together")
	}

	Configuration.ApplyChanges = *Values.ApplyChanges
	Configuration.ProjectDirectoryPath = *Values.ProjectDirectoryPath
	Configuration.InventoryPath = *Values.InventoryPath
	Configuration.TargetHosts = *Values.TargetHosts
	Configuration.Interactive = !*Values.NonInteractive && StandardInputIsTerminal()
	Configuration.VaultPasswordFilePath = *Values.VaultPasswordFilePath
	Configuration.AnsiblePlaybookPath = *Values.AnsiblePlaybookPath
	Configuration.AnsibleRuntimePath = *Values.AnsibleRuntimePath
	Configuration.BootstrapAnsible = !*Values.NoBootstrapAnsible
	if *Values.AskBecomePassword {
		Configuration.AskBecomePassword = PromptModeEnabled
	}
	if *Values.NoAskBecomePassword {
		Configuration.AskBecomePassword = PromptModeDisabled
	}
	if *Values.AskVaultPassword {
		Configuration.AskVaultPassword = PromptModeEnabled
	}
	if *Values.NoAskVaultPassword {
		Configuration.AskVaultPassword = PromptModeDisabled
	}
	return nil
}

func ParseInstallConfiguration(Arguments []string) (CLIConfiguration, error) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.Command = CLICommandInstall

	Flags := flag.NewFlagSet("smf install", flag.ContinueOnError)
	Flags.SetOutput(io.Discard)
	OperationValues := RegisterOperationFlags(Flags)
	PublishedPort := Flags.Int("port", 0, "publish this host TCP port to SMF container port 80")
	NetworkOnly := Flags.Bool("network-only", false, "keep SMF reachable only on its Docker networks")
	DisableInstaller := Flags.Bool("disable-installer", false, "disable SMF's web installer during deployment")

	if FlagError := Flags.Parse(Arguments); FlagError != nil {
		return CLIConfiguration{}, FlagError
	}
	if len(Flags.Args()) != 0 {
		return CLIConfiguration{}, fmt.Errorf("unexpected argument %q", Flags.Args()[0])
	}
	if OperationFlagsError := ApplyOperationFlags(&Configuration, OperationValues); OperationFlagsError != nil {
		return CLIConfiguration{}, OperationFlagsError
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

	Configuration.PublishedPort = *PublishedPort
	Configuration.ExposureWasSpecified = VisitedFlags["port"] || *NetworkOnly
	Configuration.InstallerEnabled = !*DisableInstaller
	if *PublishedPort != 0 {
		Configuration.ExposureMode = ExposureModePublishedPort
	}
	return Configuration, nil
}

func ParseConfigureConfiguration(Arguments []string) (CLIConfiguration, error) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.Command = CLICommandConfigure

	Flags := flag.NewFlagSet("smf configure", flag.ContinueOnError)
	Flags.SetOutput(io.Discard)
	OperationValues := RegisterOperationFlags(Flags)
	PublishedPort := Flags.Int("port", 0, "publish this host TCP port to SMF container port 80")
	NetworkOnly := Flags.Bool("network-only", false, "keep SMF reachable only on its Docker networks")
	ForumURL := Flags.String("forum-url", "", "public forum URL without a trailing slash")

	if FlagError := Flags.Parse(Arguments); FlagError != nil {
		return CLIConfiguration{}, FlagError
	}
	if len(Flags.Args()) != 0 {
		return CLIConfiguration{}, fmt.Errorf("unexpected argument %q", Flags.Args()[0])
	}
	if OperationFlagsError := ApplyOperationFlags(&Configuration, OperationValues); OperationFlagsError != nil {
		return CLIConfiguration{}, OperationFlagsError
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

	Configuration.PublishedPort = *PublishedPort
	Configuration.ExposureWasSpecified = VisitedFlags["port"] || *NetworkOnly
	Configuration.ForumURLWasSpecified = VisitedFlags["forum-url"]
	if Configuration.ForumURLWasSpecified {
		ValidatedForumURL, ForumURLError := ValidateForumURL(*ForumURL)
		if ForumURLError != nil {
			return CLIConfiguration{}, ForumURLError
		}
		Configuration.ForumURL = ValidatedForumURL
	}
	if *PublishedPort != 0 {
		Configuration.ExposureMode = ExposureModePublishedPort
	}
	if !Configuration.Interactive {
		ValidatedConfiguration, ConfigurationError := ValidateConfigureConfiguration(Configuration)
		if ConfigurationError != nil {
			return CLIConfiguration{}, ConfigurationError
		}
		Configuration = ValidatedConfiguration
	}
	return Configuration, nil
}

func ValidateConfigureConfiguration(Configuration CLIConfiguration) (CLIConfiguration, error) {
	if !Configuration.ExposureWasSpecified {
		return CLIConfiguration{}, fmt.Errorf("smf configure requires --port or --network-only when input is not interactive")
	}
	if !Configuration.ForumURLWasSpecified {
		return CLIConfiguration{}, fmt.Errorf("smf configure requires --forum-url when input is not interactive")
	}
	ValidatedForumURL, ForumURLError := ValidateForumURL(Configuration.ForumURL)
	if ForumURLError != nil {
		return CLIConfiguration{}, ForumURLError
	}
	Configuration.ForumURL = ValidatedForumURL
	return Configuration, nil
}

func ValidateForumURL(ForumURL string) (string, error) {
	if ForumURL == "" {
		return "", fmt.Errorf("--forum-url must not be empty")
	}
	if strings.TrimSpace(ForumURL) != ForumURL || strings.ContainsAny(ForumURL, " \t\r\n") {
		return "", fmt.Errorf("--forum-url must not contain whitespace")
	}
	if strings.HasSuffix(ForumURL, "/") {
		return "", fmt.Errorf("--forum-url must not end with a slash")
	}
	ParsedForumURL, ParseError := url.ParseRequestURI(ForumURL)
	if ParseError != nil {
		return "", fmt.Errorf("parse --forum-url: %w", ParseError)
	}
	if ParsedForumURL.Scheme != "http" && ParsedForumURL.Scheme != "https" {
		return "", fmt.Errorf("--forum-url must use http or https")
	}
	if ParsedForumURL.Host == "" || ParsedForumURL.Hostname() == "" {
		return "", fmt.Errorf("--forum-url must include a host")
	}
	if ParsedForumURL.User != nil {
		return "", fmt.Errorf("--forum-url must not include user credentials")
	}
	if ParsedForumURL.RawQuery != "" || ParsedForumURL.Fragment != "" {
		return "", fmt.Errorf("--forum-url must not include a query or fragment")
	}
	return ForumURL, nil
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
  smf configure [options]
  smf list [options]
  smf --version
  smf --help

Commands:
  install    Plan an SMF installation and, after confirmation or --apply, run
             playbooks/install-all.yml.
  configure  Change the deployed SMF exposure and forum URL without using
             install.php.
  list       Show the local SMF Compose service state.

Install and configure options:
  --port PORT              Publish PORT on the host as PORT:80.
  --network-only           Keep SMF on Docker networks only.
  --apply                  Apply the plan without final confirmation.
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
  --non-interactive        Do not ask for settings or confirmation.

Install-only option:
  --disable-installer      Disable SMF's web installer during this deployment.

Configure-only option:
  --forum-url URL          Public forum URL without a trailing slash. Required
                           with --non-interactive; prompted otherwise.

List option:
  --deployment-dir PATH    Local SMF Compose deployment directory (default /opt/smf).

`)
}
