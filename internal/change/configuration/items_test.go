// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"encoding/json"
	"errors"
	"net/netip"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestTheListenAddressIsTheLoopbackAddressTheConfigurationStates(t *testing.T) {
	stated := completeItems(t)
	stated[listenAddressKey] = quoted(t, "[::1]:8443")

	configured := readConfiguration(t, stated)

	if configured.ListenAddress() != netip.MustParseAddrPort("[::1]:8443") {
		t.Errorf("the listen address is %s, want [::1]:8443", configured.ListenAddress())
	}
}

func TestAListenAddressThatIsNotALoopbackAddressWithAPortIsRefusedNamingIt(t *testing.T) {
	for _, address := range []string{
		"0.0.0.0:8080", "[::]:8080", "192.0.2.10:8080", "localhost:8080",
		"127.0.0.1", "127.0.0.1:0", "127.0.0.1:http",
	} {
		t.Run(address, func(t *testing.T) {
			stated := completeItems(t)
			stated[listenAddressKey] = quoted(t, address)

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, listenAddressKey)
		})
	}
}

func TestTheInstanceURLIsTheOriginTheConfigurationStates(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))

	if configured.InstanceURL().String() != statedInstanceURL {
		t.Errorf("the instance URL is %s, want %s", configured.InstanceURL(), statedInstanceURL)
	}
}

func TestAnInstanceURLThatIsNotAnHTTPSOriginIsRefusedNamingIt(t *testing.T) {
	stated := completeItems(t)
	stated[instanceURLKey] = quoted(t, "http://control-plane.example/ci")

	assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, instanceURLKey)
}

func TestTheAppIDIsTheOneTheConfigurationStates(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))

	if configured.AppID() != statedAppID {
		t.Errorf("the App ID is %d, want %d", configured.AppID(), statedAppID)
	}
}

func TestANonPositiveAppIDIsRefusedNamingIt(t *testing.T) {
	for _, appID := range []int64{0, -statedAppID} {
		stated := completeItems(t)
		stated[appIDKey] = number(appID)

		assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, appIDKey)
	}
}

func TestTheAppKeyIsReadFromTheFileTheConfigurationNames(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))

	if string(configured.AppKey().Reveal()) != statedAppKey {
		t.Error("the App key is not the content of the file the configuration names")
	}
}

func TestTheWebhookSecretIsReadFromTheFileTheConfigurationNames(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))

	if string(configured.WebhookSecret().Reveal()) != statedWebhookSecret {
		t.Error("the webhook secret is not the content of the file the configuration names")
	}
}

func TestOneTrailingLineBreakEndsTheSecretFileAndIsNotPartOfTheSecret(t *testing.T) {
	stated := completeItems(t)
	stated[webhookSecretFileKey] = quoted(t, writtenFile(t, "webhook-secret", statedWebhookSecret+"\n"))

	configured := readConfiguration(t, stated)

	if string(configured.WebhookSecret().Reveal()) != statedWebhookSecret {
		t.Error("the trailing line break was taken as part of the webhook secret")
	}
}

func TestAWebhookSecretThatIsNotPrintableASCIIIsRefusedNamingIt(t *testing.T) {
	for form, secret := range map[string]string{
		"a lone line break": "\n",
		"a CRLF line end":   "s3cret\r\n",
		"a NUL":             "s3c\x00ret",
		"a tab":             "s3c\tret",
		"non-ASCII":         "s3crét",
	} {
		t.Run(form, func(t *testing.T) {
			stated := completeItems(t)
			stated[webhookSecretFileKey] = quoted(t, writtenFile(t, "webhook-secret", secret))

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, webhookSecretFileKey)
		})
	}
}

func TestANamedFileThatIsEmptyOrAbsentIsRefusedNamingItsItem(t *testing.T) {
	for _, key := range []string{appKeyFileKey, webhookSecretFileKey} {
		for form, path := range map[string]string{
			"empty":  writtenFile(t, "empty", ""),
			"absent": filepath.Join(t.TempDir(), "absent"),
		} {
			t.Run(key+" "+form, func(t *testing.T) {
				stated := completeItems(t)
				stated[key] = quoted(t, path)

				assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, key)
			})
		}
	}
}

func TestANamedPathThatIsNotARegularFileIsRefusedNamingItsItem(t *testing.T) {
	pipe := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatalf("make a named pipe: %v", err)
	}
	for _, key := range []string{appKeyFileKey, webhookSecretFileKey} {
		t.Run(key, func(t *testing.T) {
			stated := completeItems(t)
			stated[key] = quoted(t, pipe)

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, key)
		})
	}
}

func TestANamedFileAtItsBoundIsReadAndOneByteMoreIsRefusedNamingItsItem(t *testing.T) {
	bounds := map[string]int{
		appKeyFileKey:        configuration.MaxSecretFileBytes,
		webhookSecretFileKey: configuration.MaxSecretFileBytes,
	}
	for key, maxBytes := range bounds {
		t.Run(key, func(t *testing.T) {
			atBound := completeItems(t)
			atBound[key] = quoted(t, writtenFile(t, key, strings.Repeat("s", maxBytes)))
			readConfiguration(t, atBound)

			overBound := completeItems(t)
			overBound[key] = quoted(t, writtenFile(t, key, strings.Repeat("s", maxBytes+1)))
			err := refusalOf(t, overBound)

			assertOnlyRefusal(t, err, configuration.ErrInvalidItem, key)
			if !errors.Is(err, configuration.ErrOversizedFile) {
				t.Errorf("the refusal does not say the file is oversized: %v", err)
			}
		})
	}
}

func TestTheInstallationAllowlistIsTheOneTheConfigurationStates(t *testing.T) {
	stated := completeItems(t)
	stated[installationIDsKey] = listOf(number(statedInstallationID), number(statedInstallationID+1))

	configured := readConfiguration(t, stated)

	want := []int64{statedInstallationID, statedInstallationID + 1}
	if !slices.Equal(configured.InstallationIDs(), want) {
		t.Errorf("the allowlist is %v, want %v", configured.InstallationIDs(), want)
	}
}

func TestAnAllowlistAtTheBoundIsReadAndOneMoreIsRefused(t *testing.T) {
	atBound := completeItems(t)
	atBound[installationIDsKey] = distinctInstallationIDs(configuration.MaxInstallations)
	readConfiguration(t, atBound)

	overBound := completeItems(t)
	overBound[installationIDsKey] = distinctInstallationIDs(configuration.MaxInstallations + 1)

	assertOnlyRefusal(t, refusalOf(t, overBound), configuration.ErrInvalidItem, installationIDsKey)
}

func TestAnAllowlistHoldingANonPositiveOrRepeatedIDIsRefusedNamingIt(t *testing.T) {
	for form, ids := range map[string]json.RawMessage{
		"a zero ID":     listOf(number(0)),
		"a negative ID": listOf(number(-statedInstallationID)),
		"a repeated ID": listOf(number(statedInstallationID), number(statedInstallationID)),
	} {
		t.Run(form, func(t *testing.T) {
			stated := completeItems(t)
			stated[installationIDsKey] = ids

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, installationIDsKey)
		})
	}
}

func TestThePolicyFamiliesAreTheOnesTheConfigurationNames(t *testing.T) {
	stated := completeItems(t)
	stated[familiesKey] = listOf(
		quoted(t, string(domain.PolicyFamilyStructural)), quoted(t, string(domain.PolicyFamilySecurity)))

	configured := readConfiguration(t, stated)

	want := []domain.PolicyFamily{domain.PolicyFamilyStructural, domain.PolicyFamilySecurity}
	if !slices.Equal(configured.Families(), want) {
		t.Errorf("the families are %v, want %v", configured.Families(), want)
	}
}

func TestAFamilyThatIsUndeclaredOrRepeatedIsRefusedNamingIt(t *testing.T) {
	structural := string(domain.PolicyFamilyStructural)
	for form, named := range map[string]json.RawMessage{
		"an undeclared family": listOf(quoted(t, structural), quoted(t, "selection")),
		"a repeated family":    listOf(quoted(t, structural), quoted(t, structural)),
		"an empty name":        listOf(quoted(t, "")),
	} {
		t.Run(form, func(t *testing.T) {
			stated := completeItems(t)
			stated[familiesKey] = named

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, familiesKey)
		})
	}
}

func distinctInstallationIDs(count int) json.RawMessage {
	ids := make([][]byte, 0, count)
	for offset := range count {
		ids = append(ids, number(statedInstallationID+int64(offset)))
	}
	return listOf(ids...)
}

func TestThePolicyRepositoryAndItsInstallationAreTheOnesTheConfigurationNames(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))

	if configured.PolicyRepository().String() != statedPolicyRepository {
		t.Errorf("the policy repository is %s, want %s", configured.PolicyRepository(), statedPolicyRepository)
	}
	if configured.PolicyInstallationID() != statedInstallationID {
		t.Errorf("the policy installation is %d, want %d", configured.PolicyInstallationID(), statedInstallationID)
	}
}

func TestAPolicyRepositoryOrInstallationThatIsMalformedIsRefusedNamingIt(t *testing.T) {
	refused := map[string]json.RawMessage{
		policyRepositoryKey:   quoted(t, "open-ci-policy"),
		policyInstallationKey: number(0),
	}
	for key, value := range refused {
		t.Run(key, func(t *testing.T) {
			stated := completeItems(t)
			stated[key] = value

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrInvalidItem, key)
		})
	}
}

func TestAnAppKeyItsParserRefusesIsRefusedNamingTheAppKeyFile(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))
	unparseable := errors.New("not one PKCS#1 RSA private key")

	_, err := configuration.AppKeyParsedBy(configured, func([]byte) (string, error) { return "", unparseable })

	if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), appKeyFileKey) {
		t.Errorf("the refusal is not ErrInvalidItem naming %s: %v", appKeyFileKey, err)
	}
	if !errors.Is(err, unparseable) {
		t.Errorf("the refusal does not carry the parser's reason: %v", err)
	}
}

func TestClassificationRulesTheirParserRefusesAreRefusedNamingTheRulesFile(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))
	unparseable := errors.New("not a classification rule document")

	_, err := configuration.ClassificationRulesParsedBy(configured, func([]byte) (string, error) { return "", unparseable })

	if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), classificationRulesFileKey) {
		t.Errorf("the refusal is not ErrInvalidItem naming %s: %v", classificationRulesFileKey, err)
	}
	if !errors.Is(err, unparseable) {
		t.Errorf("the refusal does not carry the parser's reason: %v", err)
	}
}
