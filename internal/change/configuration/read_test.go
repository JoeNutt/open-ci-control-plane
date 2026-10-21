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

func TestAConfigurationWithoutTheAppKeyIsRefusedNamingIt(t *testing.T) {
	_, err := configuration.Read(writtenFile(t, `{}`))

	if !errors.Is(err, configuration.ErrMissingItem) || !strings.Contains(err.Error(), "appKeyFile") {
		t.Fatalf("refusal is not a missing appKeyFile: %v", err)
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
	stated := completeItems(t)

	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if configured.WebhookSecret() != statedWebhookSecret {
		t.Errorf("the webhook secret is not the content of %s", stated.WebhookSecretFile)
	}
}

func TestAWebhookSecretFileAboveTheBoundIsRefused(t *testing.T) {
	secretFile := writtenFile(t, strings.Repeat("s", configuration.MaxSecretFileBytes+1))

	_, err := configuration.Read(configurationFile(t, statedItems{WebhookSecretFile: secretFile}))

	if !errors.Is(err, configuration.ErrOversizedFile) {
		t.Fatalf("an oversized webhook secret file was not refused as oversized: %v", err)
	}
}

func TestAnEmptyWebhookSecretFileIsRefusedNamingIt(t *testing.T) {
	_, err := configuration.Read(configurationFile(t, statedItems{WebhookSecretFile: writtenFile(t, "")}))

	if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "webhookSecretFile") {
		t.Fatalf("refusal is not an invalid webhookSecretFile: %v", err)
	}
}

func TestOneTrailingLineBreakEndsTheSecretFileAndIsNotPartOfTheSecret(t *testing.T) {
	stated := completeItems(t)
	stated.WebhookSecretFile = writtenFile(t, string(statedWebhookSecret)+"\n")

	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if configured.WebhookSecret() != statedWebhookSecret {
		t.Error("the trailing line break was taken as part of the webhook secret")
	}
}

func TestAWebhookSecretHoldingAByteOutsidePrintableASCIIIsRefused(t *testing.T) {
	for form, secret := range map[string]string{
		"a CRLF line end": "s3cret\r\n",
		"a NUL":           "s3c\x00ret",
		"a tab":           "s3c\tret",
		"non-ASCII":       "s3crét",
	} {
		t.Run(form, func(t *testing.T) {
			_, err := configuration.Read(configurationFile(t, statedItems{WebhookSecretFile: writtenFile(t, secret)}))

			if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "webhookSecretFile") {
				t.Fatalf("refusal is not an invalid webhookSecretFile: %v", err)
			}
		})
	}
}

func TestTheAppKeyIsReadFromTheFileTheConfigurationNames(t *testing.T) {
	stated := completeItems(t)

	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(configured.AppKey()) != statedAppKey {
		t.Errorf("the App key is not the content of %s", stated.AppKeyFile)
	}
}

const statedWebhookSecret = domain.WebhookSecret("s3cret")

const statedAppKey = "-----BEGIN RSA PRIVATE KEY-----\nstated\n-----END RSA PRIVATE KEY-----\n"

func completeItems(t *testing.T) statedItems {
	t.Helper()
	return statedItems{
		WebhookSecretFile: writtenFile(t, string(statedWebhookSecret)),
		AppKeyFile:        writtenFile(t, statedAppKey),
	}
}

type statedItems struct {
	WebhookSecretFile string `json:"webhookSecretFile,omitempty"`
	AppKeyFile        string `json:"appKeyFile,omitempty"`
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
