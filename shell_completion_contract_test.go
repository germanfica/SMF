package main

import (
	"os"
	"strings"
	"testing"
)

func readShellCompletionContractFile(t *testing.T, Path string) string {
	t.Helper()
	Contents, ReadError := os.ReadFile(Path)
	if ReadError != nil {
		t.Fatal(ReadError)
	}
	return string(Contents)
}

func TestBashCompletionRegistersInstalledAndCheckoutCommands(t *testing.T) {
	Contents := readShellCompletionContractFile(t, "completions/smf.bash")
	for _, ExpectedFragment := range []string{
		"complete -F _smf_complete smf ./smf",
		"install configure list --help --version",
		"--forum-url",
		"install|configure",
		"--no-bootstrap-ansible",
		"--deployment-dir",
		"_smf_option_was_provided --port",
	} {
		if !strings.Contains(Contents, ExpectedFragment) {
			t.Fatalf("Bash completion is missing %q", ExpectedFragment)
		}
	}
}

func TestZshCompletionIsScopedToSMFCommands(t *testing.T) {
	Contents := readShellCompletionContractFile(t, "completions/_smf")
	for _, ExpectedFragment := range []string{
		"#compdef smf ./smf",
		"compdef _smf smf ./smf",
		"configure[Change deployed SMF exposure and forum URL]",
		"--forum-url=",
		"(--network-only)--port=",
		"(--port)--network-only",
		"--deployment-dir=",
	} {
		if !strings.Contains(Contents, ExpectedFragment) {
			t.Fatalf("Zsh completion is missing %q", ExpectedFragment)
		}
	}
}

func TestInstallScriptSelectsTheExpectedInstallationScopes(t *testing.T) {
	Contents := readShellCompletionContractFile(t, "install.sh")
	for _, ExpectedFragment := range []string{
		"InstallationDirectoryPath=\"/usr/local/bin\"",
		"--user",
		"--user and --bin-dir cannot be used together",
		"RunPrivileged install -m 0755",
		"/usr/local/share/bash-completion/completions/$ProgramName",
		"/usr/local/share/zsh/site-functions/_$ProgramName",
		"--no-completions",
		"completions/$ProgramName.bash",
		"completions/_$ProgramName",
		"Installed Bash completion",
		"Installed Zsh completion",
	} {
		if !strings.Contains(Contents, ExpectedFragment) {
			t.Fatalf("install.sh is missing %q", ExpectedFragment)
		}
	}
}
