package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallPlaybookRemovesInstallerAndReplacesHostPorts(t *testing.T) {
	PlaybookPath := filepath.Join("playbooks", "install-smf.yml")
	Playbook, ReadError := os.ReadFile(PlaybookPath)
	if ReadError != nil {
		t.Fatalf("read %s: %v", PlaybookPath, ReadError)
	}
	for _, RequiredFragment := range []string{
		"RUN rm -f /var/www/html/install.php",
		"ports: !override",
		"ports: !reset []",
		"test ! -e /var/www/html/install.php",
		"SMF did not publish host port",
	} {
		if !strings.Contains(string(Playbook), RequiredFragment) {
			t.Errorf("%s does not contain %q", PlaybookPath, RequiredFragment)
		}
	}
}
