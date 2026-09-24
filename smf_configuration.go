package main

import (
	"bufio"
	"fmt"
	"os"
)

func RunSMFConfiguration(Configuration CLIConfiguration) error {
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
	if Configuration.Interactive && !Configuration.ForumURLWasSpecified {
		InteractiveConfiguration, InteractiveConfigurationError := PromptForForumURL(Configuration, TerminalReader)
		if InteractiveConfigurationError != nil {
			return InteractiveConfigurationError
		}
		Configuration = InteractiveConfiguration
	}
	ValidatedConfiguration, ConfigurationError := ValidateConfigureConfiguration(Configuration)
	if ConfigurationError != nil {
		return ConfigurationError
	}
	Configuration = ValidatedConfiguration

	AskVaultPassword, AskVaultPasswordError := ShouldAskVaultPassword(Configuration, Project)
	if AskVaultPasswordError != nil {
		return AskVaultPasswordError
	}
	AnsiblePlan, AnsiblePlanError := DescribeAnsibleResolution(Configuration)
	if AnsiblePlanError != nil {
		return AnsiblePlanError
	}
	PrintSMFConfigurationPlan(Configuration, Project, AnsiblePlan, AskVaultPassword)

	if !Configuration.ApplyChanges && Configuration.Interactive {
		ApplyChanges, ConfirmationError := PromptForPlanConfirmation(TerminalReader)
		if ConfirmationError != nil {
			return ConfirmationError
		}
		Configuration.ApplyChanges = ApplyChanges
	}
	if !Configuration.ApplyChanges {
		fmt.Fprintln(os.Stdout, "\nRun again with \""+BuildSMFConfigurationApplyCommand(Configuration)+"\" to apply this plan.")
		return nil
	}

	AnsibleResolution, AnsibleResolutionError := EnsureAnsiblePlaybook(Configuration)
	if AnsibleResolutionError != nil {
		return AnsibleResolutionError
	}
	AnsibleOperation, AnsibleOperationError := BuildConfigureAnsiblePlaybookOperation(Configuration, Project, AnsibleResolution, AskVaultPassword)
	if AnsibleOperationError != nil {
		return AnsibleOperationError
	}
	fmt.Fprintln(os.Stdout, "\nExecuting:", FormatCommandOperation(AnsibleOperation))
	if ConfigurationError := ExecuteCommandOperation(AnsibleOperation); ConfigurationError != nil {
		return ConfigurationError
	}
	PrintSMFConfigurationSuccess(Configuration)
	return nil
}

func PrintSMFConfigurationPlan(Configuration CLIConfiguration, Project SMFProject, AnsiblePlan AnsibleResolution, AskVaultPassword bool) {
	fmt.Fprintln(os.Stdout, "SMF configuration plan:")
	fmt.Fprintln(os.Stdout, "- Use project:", Project.RootPath)
	if AnsiblePlan.AnsiblePlaybookPath != "" {
		fmt.Fprintln(os.Stdout, "- Use ansible-playbook:", AnsiblePlan.AnsiblePlaybookPath)
	} else {
		fmt.Fprintln(os.Stdout, "- Create a private ansible-core "+AnsibleCoreVersion+" environment:", AnsiblePlan.RuntimePath)
	}
	if Configuration.ExposureMode == ExposureModePublishedPort {
		fmt.Fprintf(os.Stdout, "- Publish host port %d to SMF container port 80\n", Configuration.PublishedPort)
	} else {
		fmt.Fprintln(os.Stdout, "- Keep SMF on its Docker networks only (internal port 80)")
	}
	fmt.Fprintln(os.Stdout, "- Set the persisted SMF forum URL to", Configuration.ForumURL)
	fmt.Fprintln(os.Stdout, "- Synchronize SMF URL settings for themes, images, smileys, and avatars")
	fmt.Fprintln(os.Stdout, "- Recreate only the SMF Compose service")
	fmt.Fprintln(os.Stdout, "- Do not run install.php or modify forum content, users, or database schema")
	if ShouldAskBecomePassword(Configuration) {
		fmt.Fprintln(os.Stdout, "- Ask Ansible once for the sudo password")
	}
	if Configuration.VaultPasswordFilePath != "" {
		fmt.Fprintln(os.Stdout, "- Use the configured Ansible Vault password file")
	} else if AskVaultPassword {
		fmt.Fprintln(os.Stdout, "- Ask Ansible for the Vault password")
	}
}

func BuildSMFConfigurationApplyCommand(Configuration CLIConfiguration) string {
	ExposureArgument := "--network-only"
	if Configuration.ExposureMode == ExposureModePublishedPort {
		ExposureArgument = fmt.Sprintf("--port %d", Configuration.PublishedPort)
	}
	return "smf configure " + ExposureArgument + " --forum-url " + QuoteShellArgument(Configuration.ForumURL) + " --apply"
}

func PrintSMFConfigurationSuccess(Configuration CLIConfiguration) {
	fmt.Fprintln(os.Stdout, "\nSMF configuration completed successfully.")
	fmt.Fprintln(os.Stdout, "Forum URL:", Configuration.ForumURL)
}
