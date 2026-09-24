package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurePlaybookReconfiguresOnlyExposureAndForumURL(t *testing.T) {
	PlaybookPath := filepath.Join("playbooks", "configure-smf.yml")
	Playbook, ReadError := os.ReadFile(PlaybookPath)
	if ReadError != nil {
		t.Fatalf("read %s: %v", PlaybookPath, ReadError)
	}
	for _, RequiredFragment := range []string{
		"Reconfigure an existing SMF deployment without rerunning install.php.",
		"--format",
		"json",
		"ports: !override",
		"ports: !reset []",
		"--force-recreate",
		"--no-deps",
		"/var/www/html/Settings.php",
		"realpath('/var/www/html/Settings.php')",
		"/var/www/smf-config",
		"SMF_FORUM_URL",
		"exactly one \\$boardurl assignment",
		"scripts/configure-smf-url-settings.php",
		"Synchronize the persisted SMF URL configuration",
		"SMF did not publish host port",
	} {
		if !strings.Contains(string(Playbook), RequiredFragment) {
			t.Errorf("%s does not contain %q", PlaybookPath, RequiredFragment)
		}
	}
}

func TestConfigureURLSettingsScriptUpdatesOnlyManagedConfiguration(t *testing.T) {
	ScriptPath := filepath.Join("scripts", "configure-smf-url-settings.php")
	Script, ReadError := os.ReadFile(ScriptPath)
	if ReadError != nil {
		t.Fatalf("read %s: %v", ScriptPath, ReadError)
	}
	for _, RequiredFragment := range []string{
		"avatar_url",
		"custom_avatar_url",
		"smileys_url",
		"theme_url",
		"images_url",
		"base_theme_url",
		"base_images_url",
		"id_member = {int:global_member}",
		"UPDATE {db_prefix}settings",
		"UPDATE {db_prefix}themes",
		"settings_updated",
		"cache_put_data('modSettings', null, 90)",
		"External CDNs use",
	} {
		if !strings.Contains(string(Script), RequiredFragment) {
			t.Errorf("%s does not contain %q", ScriptPath, RequiredFragment)
		}
	}
}

func TestConfigurePlaybookPassesChdirToTheCommandModule(t *testing.T) {
	PlaybookPath := filepath.Join("playbooks", "configure-smf.yml")
	Playbook, ReadError := os.ReadFile(PlaybookPath)
	if ReadError != nil {
		t.Fatalf("read %s: %v", PlaybookPath, ReadError)
	}
	for _, InvalidTaskLevelChdir := range []string{
		"\n      chdir: \"{{ SMF_BASE_PATH }}\"",
		"\n          chdir: \"{{ SMF_BASE_PATH }}\"",
	} {
		if strings.Contains(string(Playbook), InvalidTaskLevelChdir) {
			t.Errorf("%s places chdir beside ansible.builtin.command", PlaybookPath)
		}
	}
}
