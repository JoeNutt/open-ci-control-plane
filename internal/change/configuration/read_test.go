// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAConfigurationMissingAnItemIsRefusedNamingThatItem(t *testing.T) {
	removals := map[string]func(*statedItems){
		"webhookSecretFile": func(stated *statedItems) { stated.WebhookSecretFile = "" },
		"appKeyFile":        func(stated *statedItems) { stated.AppKeyFile = "" },
		"installationIds":   func(stated *statedItems) { stated.InstallationIDs = nil },
		"families":          func(stated *statedItems) { stated.Families = nil },
		"listenAddress":     func(stated *statedItems) { stated.ListenAddress = "" },
		"appId":             func(stated *statedItems) { stated.AppID = nil },
	}
	for item, remove := range removals {
		t.Run(item, func(t *testing.T) {
			stated := completeItems(t)
			remove(&stated)

			_, err := configuration.Read(configurationFile(t, stated))

			assertRefusedNaming(t, err, configuration.ErrMissingItem, item)
		})
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

	assertRefusedNaming(t, err, configuration.ErrInvalidItem, "webhookSecretFile")
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

			assertRefusedNaming(t, err, configuration.ErrInvalidItem, "webhookSecretFile")
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

			assertRefusedNaming(t, err, configuration.ErrInvalidItem, "installationIds")
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

			assertRefusedNaming(t, err, configuration.ErrInvalidItem, "families")
		})
	}
}

func TestTheListenAddressIsTheLoopbackAddressTheConfigurationStates(t *testing.T) {
	stated := completeItems(t)
	stated.ListenAddress = "[::1]:8443"

	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if configured.ListenAddress() != netip.MustParseAddrPort(stated.ListenAddress) {
		t.Errorf("the listen address is %s, want %s", configured.ListenAddress(), stated.ListenAddress)
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

			assertRefusedNaming(t, err, configuration.ErrInvalidItem, "listenAddress")
		})
	}
}

func TestANonPositiveAppIDIsRefusedNamingIt(t *testing.T) {
	for _, stated := range []int64{0, -statedAppID} {
		t.Run(fmt.Sprint(stated), func(t *testing.T) {
			items := completeItems(t)
			items.AppID = &stated

			_, err := configuration.Read(configurationFile(t, items))

			assertRefusedNaming(t, err, configuration.ErrInvalidItem, "appId")
		})
	}
}

func TestAnAppKeyFileAboveTheBoundIsRefusedNamingIt(t *testing.T) {
	stated := completeItems(t)
	stated.AppKeyFile = writtenFile(t, strings.Repeat("k", configuration.MaxAppKeyFileBytes+1))

	_, err := configuration.Read(configurationFile(t, stated))

	assertRefusedNaming(t, err, configuration.ErrOversizedFile, "appKeyFile")
}

func TestAnEmptyAppKeyFileIsRefusedNamingIt(t *testing.T) {
	stated := completeItems(t)
	stated.AppKeyFile = writtenFile(t, "")

	_, err := configuration.Read(configurationFile(t, stated))

	assertRefusedNaming(t, err, configuration.ErrInvalidItem, "appKeyFile")
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

const statedAppID = 1_204_331

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
		AppID:             new(int64(statedAppID)),
	}
}

type statedItems struct {
	WebhookSecretFile string   `json:"webhookSecretFile,omitempty"`
	AppKeyFile        string   `json:"appKeyFile,omitempty"`
	InstallationIDs   []int64  `json:"installationIds,omitempty"`
	Families          []string `json:"families,omitempty"`
	ListenAddress     string   `json:"listenAddress,omitempty"`
	AppID             *int64   `json:"appId,omitempty"`
}

func configurationFile(t *testing.T, stated statedItems) string {
	t.Helper()
	content, err := json.Marshal(stated)
	if err != nil {
		t.Fatalf("encode the configuration: %v", err)
	}
	return writtenFile(t, string(content))
}

func assertRefusedNaming(t *testing.T, err error, refusal error, item string) {
	t.Helper()
	if !errors.Is(err, refusal) || !strings.Contains(err.Error(), item) {
		t.Fatalf("refusal is not %v naming %s: %v", refusal, item, err)
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
