// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAConfigurationWithoutTheWebhookSecretIsRefusedNamingIt(t *testing.T) {
	_, err := configuration.Read(writtenFile(t, `{}`))

	if !errors.Is(err, configuration.ErrMissingItem) || !strings.Contains(err.Error(), "webhookSecretFile") {
		t.Fatalf("refusal is not a missing webhookSecretFile: %v", err)
	}
}

func TestAConfigurationFileAboveTheBoundIsRefusedWhateverItSays(t *testing.T) {
	stated := `{"webhookSecretFile":"secret"}`
	padded := stated + strings.Repeat(" ", configuration.MaxFileBytes+1-len(stated))

	_, err := configuration.Read(writtenFile(t, padded))

	if !errors.Is(err, configuration.ErrOversizedFile) {
		t.Fatalf("a %d-byte configuration was not refused as oversized: %v", len(padded), err)
	}
}

func TestTheWebhookSecretIsReadFromTheFileTheConfigurationNames(t *testing.T) {
	secretFile := writtenFile(t, "s3cret")

	read, err := configuration.Read(writtenFile(t, `{"webhookSecretFile":"`+secretFile+`"}`))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if read.WebhookSecret() != domain.WebhookSecret("s3cret") {
		t.Errorf("the webhook secret is not the content of %s", secretFile)
	}
}

func writtenFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "controlplane.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
