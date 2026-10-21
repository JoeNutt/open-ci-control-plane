// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func TestAConfigurationWithoutTheInstallationAllowlistIsRefusedNamingIt(t *testing.T) {
	_, err := configuration.Read(writtenFile(t, `{}`))

	if !errors.Is(err, configuration.ErrMissingItem) || !strings.Contains(err.Error(), "installationIds") {
		t.Fatalf("refusal is not a missing installationIds: %v", err)
	}
}

func TestAConfigurationNamingNoPolicyFamilyIsRefusedNamingIt(t *testing.T) {
	_, err := configuration.Read(writtenFile(t, `{}`))

	if !errors.Is(err, configuration.ErrMissingItem) || !strings.Contains(err.Error(), "families") {
		t.Fatalf("refusal is not a missing families: %v", err)
	}
}

func TestAConfigurationWithoutAListenAddressIsRefusedNamingIt(t *testing.T) {
	_, err := configuration.Read(writtenFile(t, `{}`))

	if !errors.Is(err, configuration.ErrMissingItem) || !strings.Contains(err.Error(), "listenAddress") {
		t.Fatalf("refusal is not a missing listenAddress: %v", err)
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

	if string(configured.AppKey().Reveal()) != statedAppKey {
		t.Errorf("the App key is not the content of %s", stated.AppKeyFile)
	}
}

func TestTheInstallationAllowlistIsTheOneTheConfigurationStates(t *testing.T) {
	stated := completeItems(t)
	stated.InstallationIDs = []int64{statedInstallationID, statedInstallationID + 1}

	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !slices.Equal(configured.InstallationIDs(), stated.InstallationIDs) {
		t.Errorf("the allowlist is %v, want %v", configured.InstallationIDs(), stated.InstallationIDs)
	}
}

func TestAnInvalidInstallationAllowlistIsRefusedNamingIt(t *testing.T) {
	refused := map[string][]int64{
		"a zero ID":           {0},
		"a negative ID":       {-statedInstallationID},
		"a repeated ID":       {statedInstallationID, statedInstallationID},
		"more than the bound": slices.Repeat([]int64{statedInstallationID}, configuration.MaxInstallations+1),
	}
	for form, ids := range refused {
		t.Run(form, func(t *testing.T) {
			stated := completeItems(t)
			stated.InstallationIDs = ids

			_, err := configuration.Read(configurationFile(t, stated))

			if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "installationIds") {
				t.Fatalf("refusal is not an invalid installationIds: %v", err)
			}
		})
	}
}

func TestThePolicyFamiliesAreTheOnesTheConfigurationNames(t *testing.T) {
	stated := completeItems(t)
	stated.Families = []string{string(domain.PolicyFamilyStructural), string(domain.PolicyFamilySecurity)}

	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	want := []domain.PolicyFamily{domain.PolicyFamilyStructural, domain.PolicyFamilySecurity}
	if !slices.Equal(configured.Families(), want) {
		t.Errorf("the families are %v, want %v", configured.Families(), want)
	}
}

func TestAnUnknownOrRepeatedPolicyFamilyIsRefusedNamingIt(t *testing.T) {
	structural := string(domain.PolicyFamilyStructural)
	for form, named := range map[string][]string{
		"an unknown family": {structural, "selection"},
		"a repeated family": {structural, structural},
		"an empty name":     {""},
	} {
		t.Run(form, func(t *testing.T) {
			stated := completeItems(t)
			stated.Families = named

			_, err := configuration.Read(configurationFile(t, stated))

			if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "families") {
				t.Fatalf("refusal is not an invalid families: %v", err)
			}
		})
	}
}

func TestAListenAddressOffLoopbackOrWithoutAPortIsRefusedNamingIt(t *testing.T) {
	for _, address := range []string{
		"0.0.0.0:8080",
		"[::]:8080",
		"192.0.2.10:8080",
		"localhost:8080",
		"127.0.0.1",
		"127.0.0.1:0",
		"127.0.0.1:http",
	} {
		t.Run(address, func(t *testing.T) {
			stated := completeItems(t)
			stated.ListenAddress = address

			_, err := configuration.Read(configurationFile(t, stated))

			if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "listenAddress") {
				t.Fatalf("refusal is not an invalid listenAddress: %v", err)
			}
		})
	}
}

func TestAnAppKeyFileAboveTheBoundIsRefusedNamingIt(t *testing.T) {
	stated := completeItems(t)
	stated.AppKeyFile = writtenFile(t, strings.Repeat("k", configuration.MaxAppKeyFileBytes+1))

	_, err := configuration.Read(configurationFile(t, stated))

	if !errors.Is(err, configuration.ErrOversizedFile) || !strings.Contains(err.Error(), "appKeyFile") {
		t.Fatalf("refusal is not an oversized appKeyFile: %v", err)
	}
}

func TestAnEmptyAppKeyFileIsRefusedNamingIt(t *testing.T) {
	stated := completeItems(t)
	stated.AppKeyFile = writtenFile(t, "")

	_, err := configuration.Read(configurationFile(t, stated))

	if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "appKeyFile") {
		t.Fatalf("refusal is not an invalid appKeyFile: %v", err)
	}
}

func TestNoFormattingVerbPrintsASecretTheConfigurationHolds(t *testing.T) {
	configured, err := configuration.Read(configurationFile(t, completeItems(t)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	held := map[string]any{
		"the configuration":              configured,
		"a pointer to the configuration": &configured,
		"the App key":                    configured.AppKey(),
	}

	for name, value := range held {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%d", "%x", "%X", "%q", "%p", "%+d", "%#x"} {
			printed := fmt.Sprintf(verb, value)
			for _, form := range printedFormsOf(string(statedWebhookSecret), statedAppKeyBody) {
				if strings.Contains(printed, form) {
					t.Errorf("%s under %s prints a secret as %q: %s", name, verb, form, printed)
				}
			}
		}
	}
}

func printedFormsOf(secrets ...string) []string {
	var forms []string
	for _, secret := range secrets {
		decimal := fmt.Sprintf("%d", []byte(secret))
		forms = append(forms,
			secret,
			hex.EncodeToString([]byte(secret)),
			strings.ToUpper(hex.EncodeToString([]byte(secret))),
			strings.Trim(decimal, "[]"),
		)
	}
	return forms
}

const statedListenAddress = "127.0.0.1:8080"

const statedInstallationID = 52_311_907

const statedWebhookSecret = domain.WebhookSecret("s3cret")

const statedAppKeyBody = "c3RhdGVkIGtleQ"

const statedAppKey = "-----BEGIN RSA PRIVATE KEY-----\n" + statedAppKeyBody + "\n-----END RSA PRIVATE KEY-----\n"

func completeItems(t *testing.T) statedItems {
	t.Helper()
	return statedItems{
		WebhookSecretFile: writtenFile(t, string(statedWebhookSecret)),
		AppKeyFile:        writtenFile(t, statedAppKey),
		InstallationIDs:   []int64{statedInstallationID},
		Families:          []string{string(domain.PolicyFamilyStructural)},
		ListenAddress:     statedListenAddress,
	}
}

type statedItems struct {
	WebhookSecretFile string   `json:"webhookSecretFile,omitempty"`
	AppKeyFile        string   `json:"appKeyFile,omitempty"`
	InstallationIDs   []int64  `json:"installationIds,omitempty"`
	Families          []string `json:"families,omitempty"`
	ListenAddress     string   `json:"listenAddress,omitempty"`
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
