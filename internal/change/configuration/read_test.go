// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"encoding/json"
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

	configured, err := configuration.Read(configurationFile(t, statedItems{WebhookSecretFile: secretFile}))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if configured.WebhookSecret() != domain.WebhookSecret("s3cret") {
		t.Errorf("the webhook secret is not the content of %s", secretFile)
	}
}

func TestAWebhookSecretFileAboveTheBoundIsRefused(t *testing.T) {
	secretFile := writtenFile(t, strings.Repeat("s", configuration.MaxSecretFileBytes+1))

	_, err := configuration.Read(configurationFile(t, statedItems{WebhookSecretFile: secretFile}))

	if !errors.Is(err, configuration.ErrOversizedFile) {
		t.Fatalf("an oversized webhook secret file was not refused as oversized: %v", err)
	}
}

type statedItems struct {
	WebhookSecretFile string `json:"webhookSecretFile,omitempty"`
}

func configurationFile(t *testing.T, stated statedItems) string {
	t.Helper()
	content, err := json.Marshal(stated)
	if err != nil {
		t.Fatalf("encode the configuration: %v", err)
	}
	return writtenFile(t, string(content))
}

func writtenFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "controlplane.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
