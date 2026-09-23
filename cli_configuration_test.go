package main

import "testing"

func TestParseLegacyInstallUsesApplyAndNetworkOnlyExposure(t *testing.T) {
	Configuration, ConfigurationError := ParseCLIConfiguration([]string{"--install"})
	if ConfigurationError != nil {
		t.Fatal(ConfigurationError)
	}
	if Configuration.Command != CLICommandInstall {
		t.Fatalf("command = %q, want %q", Configuration.Command, CLICommandInstall)
	}
	if !Configuration.ApplyChanges {
		t.Fatal("legacy --install must apply changes")
	}
	if Configuration.ExposureMode != ExposureModeNetworkOnly || !Configuration.ExposureWasSpecified {
		t.Fatal("legacy --install must use non-interactive network-only exposure")
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
