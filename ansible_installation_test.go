package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestBuildAnsiblePlaybookOperationPassesStructuredExtraVariables(t *testing.T) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.ExposureMode = ExposureModePublishedPort
	Configuration.PublishedPort = 8080
	Configuration.TargetHosts = "smf"
	Configuration.AskBecomePassword = PromptModeDisabled
	Project := SMFProject{
		RootPath:            "/workspace/smf",
		InventoryPath:       "/workspace/smf/inventory/hosts.yml",
		InstallPlaybookPath: "/workspace/smf/playbooks/install-all.yml",
	}
	Resolution := AnsibleResolution{AnsiblePlaybookPath: "/workspace/bin/ansible-playbook"}
	Operation, OperationError := BuildAnsiblePlaybookOperation(Configuration, Project, Resolution, false)
	if OperationError != nil {
		t.Fatal(OperationError)
	}
	if Operation.WorkingDirectory != Project.RootPath {
		t.Fatalf("working directory = %q, want %q", Operation.WorkingDirectory, Project.RootPath)
	}
	ExtraVariablesJSON := FindArgumentAfter(t, Operation.Arguments, "--extra-vars")
	ExtraVariables := make(map[string]interface{})
	if JSONError := json.Unmarshal([]byte(ExtraVariablesJSON), &ExtraVariables); JSONError != nil {
		t.Fatal(JSONError)
	}
	if ExtraVariables["SMF_PUBLISHED_PORT"] != "8080" {
		t.Fatalf("SMF_PUBLISHED_PORT = %#v, want 8080", ExtraVariables["SMF_PUBLISHED_PORT"])
	}
	if ExtraVariables["SMF_DOCKER_COMMAND_BECOME"] != true {
		t.Fatalf("SMF_DOCKER_COMMAND_BECOME = %#v, want true", ExtraVariables["SMF_DOCKER_COMMAND_BECOME"])
	}
	if ExtraVariables["TARGET_HOSTS"] != "smf" {
		t.Fatalf("TARGET_HOSTS = %#v, want smf", ExtraVariables["TARGET_HOSTS"])
	}
	if Operation.Arguments[len(Operation.Arguments)-1] != filepath.Clean(Project.InstallPlaybookPath) {
		t.Fatalf("playbook = %q, want %q", Operation.Arguments[len(Operation.Arguments)-1], Project.InstallPlaybookPath)
	}
}

func TestBuildConfigureAnsiblePlaybookOperationPassesForumURL(t *testing.T) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.ExposureMode = ExposureModePublishedPort
	Configuration.PublishedPort = 8093
	Configuration.ForumURL = "http://localhost:8093"
	Configuration.AskBecomePassword = PromptModeDisabled
	Project := SMFProject{
		RootPath:              "/workspace/smf",
		InventoryPath:         "/workspace/smf/inventory/hosts.yml",
		ConfigurePlaybookPath: "/workspace/smf/playbooks/configure-smf.yml",
	}
	Resolution := AnsibleResolution{AnsiblePlaybookPath: "/workspace/bin/ansible-playbook"}
	Operation, OperationError := BuildConfigureAnsiblePlaybookOperation(Configuration, Project, Resolution, false)
	if OperationError != nil {
		t.Fatal(OperationError)
	}
	ExtraVariablesJSON := FindArgumentAfter(t, Operation.Arguments, "--extra-vars")
	ExtraVariables := make(map[string]interface{})
	if JSONError := json.Unmarshal([]byte(ExtraVariablesJSON), &ExtraVariables); JSONError != nil {
		t.Fatal(JSONError)
	}
	if ExtraVariables["SMF_PUBLISHED_PORT"] != "8093" {
		t.Fatalf("SMF_PUBLISHED_PORT = %#v, want 8093", ExtraVariables["SMF_PUBLISHED_PORT"])
	}
	if ExtraVariables["SMF_FORUM_URL"] != "http://localhost:8093" {
		t.Fatalf("SMF_FORUM_URL = %#v", ExtraVariables["SMF_FORUM_URL"])
	}
	if Operation.Arguments[len(Operation.Arguments)-1] != filepath.Clean(Project.ConfigurePlaybookPath) {
		t.Fatalf("playbook = %q, want %q", Operation.Arguments[len(Operation.Arguments)-1], Project.ConfigurePlaybookPath)
	}
}

func FindArgumentAfter(t *testing.T, Arguments []string, Flag string) string {
	t.Helper()
	for ArgumentIndex, Argument := range Arguments {
		if Argument == Flag && ArgumentIndex+1 < len(Arguments) {
			return Arguments[ArgumentIndex+1]
		}
	}
	t.Fatalf("%q is missing from arguments %#v", Flag, Arguments)
	return ""
}
