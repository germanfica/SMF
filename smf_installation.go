package main

import (
	"bufio"
	"fmt"
	"os"
)

func RunCLI(Configuration CLIConfiguration) error {
	switch Configuration.Command {
	case CLICommandHelp:
		PrintUsage(os.Stdout)
		return nil
	case CLICommandVersion:
		fmt.Fprintln(os.Stdout, "smf", Version)
		return nil
	case CLICommandInstall:
		return RunSMFInstallation(Configuration)
	case CLICommandConfigure:
		return RunSMFConfiguration(Configuration)
	case CLICommandList:
		return RunSMFList(Configuration)
	default:
		return fmt.Errorf("unsupported command %q", Configuration.Command)
	}
}

func RunSMFInstallation(Configuration CLIConfiguration) error {
	Project, ProjectError := ResolveSMFProject(Configuration)
	if ProjectError != nil {
		return ProjectError
	}
	TerminalReader := bufio.NewReader(os.Stdin)
	if Configuration.Interactive && !Configuration.ExposureWasSpecified {
		InteractiveConfiguration, InteractiveConfigurationError := PromptForExposure(Configuration, TerminalReader)
		if InteractiveConfigurationError != nil {
			return InteractiveConfigurationError
		}
		Configuration = InteractiveConfiguration
	}
	AskVaultPassword, AskVaultPasswordError := ShouldAskVaultPassword(Configuration, Project)
	if AskVaultPasswordError != nil {
		return AskVaultPasswordError
	}
	AnsiblePlan, AnsiblePlanError := DescribeAnsibleResolution(Configuration)
	if AnsiblePlanError != nil {
		return AnsiblePlanError
	}
	PrintSMFInstallationPlan(Configuration, Project, AnsiblePlan, AskVaultPassword)

	if !Configuration.ApplyChanges && Configuration.Interactive {
		ApplyChanges, ConfirmationError := PromptForPlanConfirmation(TerminalReader)
		if ConfirmationError != nil {
			return ConfirmationError
		}
		Configuration.ApplyChanges = ApplyChanges
	}
	if !Configuration.ApplyChanges {
		fmt.Fprintln(os.Stdout, "\nRun again with \"smf install --apply\" to apply this plan.")
		return nil
	}

	AnsibleResolution, AnsibleResolutionError := EnsureAnsiblePlaybook(Configuration)
	if AnsibleResolutionError != nil {
		return AnsibleResolutionError
	}
	AnsibleOperation, AnsibleOperationError := BuildAnsiblePlaybookOperation(Configuration, Project, AnsibleResolution, AskVaultPassword)
	if AnsibleOperationError != nil {
		return AnsibleOperationError
	}
	fmt.Fprintln(os.Stdout, "\nExecuting:", FormatCommandOperation(AnsibleOperation))
	if InstallationError := ExecuteCommandOperation(AnsibleOperation); InstallationError != nil {
		return InstallationError
	}
	PrintSMFInstallationSuccess(Configuration)
	return nil
}

func PrintSMFInstallationPlan(Configuration CLIConfiguration, Project SMFProject, AnsiblePlan AnsibleResolution, AskVaultPassword bool) {
	fmt.Fprintln(os.Stdout, "SMF installation plan:")
	fmt.Fprintln(os.Stdout, "- Use project:", Project.RootPath)
	if AnsiblePlan.AnsiblePlaybookPath != "" {
		fmt.Fprintln(os.Stdout, "- Use ansible-playbook:", AnsiblePlan.AnsiblePlaybookPath)
	} else {
		fmt.Fprintln(os.Stdout, "- Create a private ansible-core "+AnsibleCoreVersion+" environment:", AnsiblePlan.RuntimePath)
	}
	fmt.Fprintln(os.Stdout, "- Install Docker Engine and its Compose plugin if needed")
	fmt.Fprintln(os.Stdout, "- Build and verify the pinned SMF image")
	if Configuration.ExposureMode == ExposureModePublishedPort {
		fmt.Fprintf(os.Stdout, "- Publish host port %d to SMF container port 80\n", Configuration.PublishedPort)
	} else {
		fmt.Fprintln(os.Stdout, "- Keep SMF on its Docker networks only (internal port 80)")
	}
	if Configuration.InstallerEnabled {
		fmt.Fprintln(os.Stdout, "- Enable the SMF web installer for initial setup")
	} else {
		fmt.Fprintln(os.Stdout, "- Disable the SMF web installer")
	}
	if ShouldAskBecomePassword(Configuration) {
		fmt.Fprintln(os.Stdout, "- Ask Ansible once for the sudo password")
	}
	if Configuration.VaultPasswordFilePath != "" {
		fmt.Fprintln(os.Stdout, "- Use the configured Ansible Vault password file")
	} else if AskVaultPassword {
		fmt.Fprintln(os.Stdout, "- Ask Ansible for the Vault password")
	}
}

func PrintSMFInstallationSuccess(Configuration CLIConfiguration) {
	fmt.Fprintln(os.Stdout, "\nSMF deployment completed successfully.")
	if !Configuration.InstallerEnabled {
		return
	}
	FinalizeCommand := "smf install --disable-installer --apply"
	if Configuration.ExposureMode == ExposureModePublishedPort {
		FinalizeCommand = fmt.Sprintf("smf install --disable-installer --port %d --apply", Configuration.PublishedPort)
	}
	fmt.Fprintln(os.Stdout, "Complete SMF's web setup, then run:")
	fmt.Fprintln(os.Stdout, "  "+FinalizeCommand)
}
