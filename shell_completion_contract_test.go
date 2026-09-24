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
		"install list --help --version",
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
		"(--network-only)--port=",
		"(--port)--network-only",
		"--deployment-dir=",
	} {
		if !strings.Contains(Contents, ExpectedFragment) {
			t.Fatalf("Zsh completion is missing %q", ExpectedFragment)
		}
	}
}

func TestInstallerCopiesBothCompletionScriptsByDefault(t *testing.T) {
	Contents := readShellCompletionContractFile(t, "installer.sh")
	for _, ExpectedFragment := range []string{
		"--no-completions",
		"completions/$ProgramName.bash",
		"completions/_$ProgramName",
		"Installed Bash completion",
		"Installed Zsh completion",
	} {
		if !strings.Contains(Contents, ExpectedFragment) {
			t.Fatalf("installer is missing %q", ExpectedFragment)
		}
	}
}
