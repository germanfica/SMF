package main

import "testing"

func TestParseInstallRejectsRemovedInstallAlias(t *testing.T) {
	_, ConfigurationError := ParseCLIConfiguration([]string{"--install"})
	if ConfigurationError == nil {
		t.Fatal("--install must not be accepted")
	}
}

func TestParseInstallPort(t *testing.T) {
	Configuration, ConfigurationError := ParseCLIConfiguration([]string{"install", "--port", "8080", "--non-interactive"})
	if ConfigurationError != nil {
		t.Fatal(ConfigurationError)
	}
	if Configuration.ExposureMode != ExposureModePublishedPort {
		t.Fatalf("exposure mode = %q, want %q", Configuration.ExposureMode, ExposureModePublishedPort)
	}
	if Configuration.PublishedPort != 8080 {
		t.Fatalf("published port = %d, want 8080", Configuration.PublishedPort)
	}
	if Configuration.Interactive {
		t.Fatal("--non-interactive must disable prompts")
	}
}

func TestParseInstallDisablesInstallerWithSelectedPort(t *testing.T) {
	Configuration, ConfigurationError := ParseCLIConfiguration([]string{"install", "--disable-installer", "--port", "8090", "--apply"})
	if ConfigurationError != nil {
		t.Fatal(ConfigurationError)
	}
	if Configuration.InstallerEnabled {
		t.Fatal("--disable-installer must disable the web installer")
	}
	if Configuration.ExposureMode != ExposureModePublishedPort {
		t.Fatalf("exposure mode = %q, want %q", Configuration.ExposureMode, ExposureModePublishedPort)
	}
	if Configuration.PublishedPort != 8090 {
		t.Fatalf("published port = %d, want 8090", Configuration.PublishedPort)
	}
	if !Configuration.ApplyChanges {
		t.Fatal("--apply must enable deployment")
	}
}

func TestParseInstallRejectsInvalidPort(t *testing.T) {
	_, ConfigurationError := ParseCLIConfiguration([]string{"install", "--port", "65536"})
	if ConfigurationError == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestParseInstallRejectsConflictingExposureOptions(t *testing.T) {
	_, ConfigurationError := ParseCLIConfiguration([]string{"install", "--network-only", "--port", "8080"})
	if ConfigurationError == nil {
		t.Fatal("expected conflicting exposure option error")
	}
}

func TestParseConfigurePortAndForumURL(t *testing.T) {
	Configuration, ConfigurationError := ParseCLIConfiguration([]string{
		"configure",
		"--port", "8093",
		"--forum-url", "http://localhost:8093",
		"--non-interactive",
	})
	if ConfigurationError != nil {
		t.Fatal(ConfigurationError)
	}
	if Configuration.Command != CLICommandConfigure {
		t.Fatalf("command = %q, want %q", Configuration.Command, CLICommandConfigure)
	}
	if Configuration.ExposureMode != ExposureModePublishedPort || Configuration.PublishedPort != 8093 {
		t.Fatalf("exposure = %q:%d, want published port 8093", Configuration.ExposureMode, Configuration.PublishedPort)
	}
	if Configuration.ForumURL != "http://localhost:8093" {
		t.Fatalf("forum URL = %q", Configuration.ForumURL)
	}
	if !Configuration.ForumURLWasSpecified {
		t.Fatal("--forum-url must be recorded as specified")
	}
}

func TestParseConfigureNetworkOnlyAndForumURL(t *testing.T) {
	Configuration, ConfigurationError := ParseCLIConfiguration([]string{
		"configure",
		"--network-only",
		"--forum-url", "https://forum.example.com",
		"--non-interactive",
	})
	if ConfigurationError != nil {
		t.Fatal(ConfigurationError)
	}
	if Configuration.ExposureMode != ExposureModeNetworkOnly {
		t.Fatalf("exposure mode = %q, want %q", Configuration.ExposureMode, ExposureModeNetworkOnly)
	}
	if !Configuration.ExposureWasSpecified {
		t.Fatal("--network-only must be recorded as specified")
	}
}

func TestParseConfigureRequiresSettingsWhenNonInteractive(t *testing.T) {
	_, ConfigurationError := ParseCLIConfiguration([]string{"configure", "--network-only", "--non-interactive"})
	if ConfigurationError == nil {
		t.Fatal("expected --forum-url requirement")
	}
}

func TestParseConfigureRequiresExposureWhenNonInteractive(t *testing.T) {
	_, ConfigurationError := ParseCLIConfiguration([]string{
		"configure",
		"--forum-url", "https://forum.example.com",
		"--non-interactive",
	})
	if ConfigurationError == nil {
		t.Fatal("expected --port or --network-only requirement")
	}
}

func TestParseConfigureRejectsTrailingSlashInForumURL(t *testing.T) {
	_, ConfigurationError := ParseCLIConfiguration([]string{
		"configure",
		"--network-only",
		"--forum-url", "https://forum.example.com/",
		"--non-interactive",
	})
	if ConfigurationError == nil {
		t.Fatal("expected trailing slash validation error")
	}
}

func TestBuildSMFConfigurationApplyCommandIncludesExposureAndForumURL(t *testing.T) {
	Configuration := NewDefaultCLIConfiguration()
	Configuration.ExposureMode = ExposureModePublishedPort
	Configuration.PublishedPort = 8093
	Configuration.ForumURL = "http://localhost:8093"
	ApplyCommand := BuildSMFConfigurationApplyCommand(Configuration)
	ExpectedCommand := "smf configure --port 8093 --forum-url http://localhost:8093 --apply"
	if ApplyCommand != ExpectedCommand {
		t.Fatalf("apply command = %q, want %q", ApplyCommand, ExpectedCommand)
	}
}
