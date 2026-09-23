package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlaybookCalculatesImageChecksumWithSHA256Sum(t *testing.T) {
	PlaybookPath := filepath.Join("playbooks", "build-smf-image.yml")
	Playbook, ReadError := os.ReadFile(PlaybookPath)
	if ReadError != nil {
		t.Fatalf("read %s: %v", PlaybookPath, ReadError)
	}
	for _, RequiredFragment := range []string{
		"- sha256sum",
		"Record the invoking user UID",
		"Restore build-user access to the temporary SMF image archive",
		"owner: \"{{ smf_build_user_uid.stdout }}\"",
		"smf_temporary_image_tar_checksum.stdout.split()[0]",
		"sha256sum did not return a valid checksum",
	} {
		if !strings.Contains(string(Playbook), RequiredFragment) {
			t.Errorf("%s does not contain %q", PlaybookPath, RequiredFragment)
		}
	}
	if strings.Contains(string(Playbook), "smf_temporary_image_tar_stat.stat.checksum") {
		t.Errorf("%s must not depend on the stat checksum field", PlaybookPath)
	}
}
