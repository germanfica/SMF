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
