// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	listenAddressKey     = "listenAddress"
	appIDKey             = "appId"
	appKeyFileKey        = "appKeyFile"
	webhookSecretFileKey = "webhookSecretFile"
	installationIDsKey   = "installationIds"
	familiesKey          = "families"
)

const (
	statedListenAddress  = "127.0.0.1:8080"
	statedAppID          = 1_204_331
	statedInstallationID = 52_311_907
	statedWebhookSecret  = "s3cret"
	statedAppKeyBody     = "c3RhdGVkIGtleQ"
	statedAppKey         = "-----BEGIN RSA PRIVATE KEY-----\n" + statedAppKeyBody + "\n-----END RSA PRIVATE KEY-----\n"
)

func TestAConfigurationMissingAnItemIsRefusedNamingThatItem(t *testing.T) {
	for _, key := range []string{
		listenAddressKey, appIDKey, appKeyFileKey, webhookSecretFileKey, installationIDsKey, familiesKey,
	} {
		t.Run(key, func(t *testing.T) {
			stated := completeItems(t)
			delete(stated, key)

			assertOnlyRefusal(t, refusalOf(t, stated), configuration.ErrMissingItem, key)
		})
	}
}

func TestAnItemStatedEmptyNullOrAsTheWrongTypeIsRefusedAsInvalidNamingIt(t *testing.T) {
	for _, stated := range []struct{ key, value string }{
		{listenAddressKey, `""`},
		{listenAddressKey, `8080`},
		{appIDKey, `"1204331"`},
		{appIDKey, `1.5`},
		{appIDKey, `null`},
		{appKeyFileKey, `""`},
		{webhookSecretFileKey, `""`},
		{webhookSecretFileKey, `["s3cret"]`},
		{installationIDsKey, `[]`},
		{installationIDsKey, `["52311907"]`},
		{familiesKey, `[]`},
		{familiesKey, `"structural"`},
	} {
		t.Run(stated.key+" "+stated.value, func(t *testing.T) {
			items := completeItems(t)
			items[stated.key] = json.RawMessage(stated.value)

			assertOnlyRefusal(t, refusalOf(t, items), configuration.ErrInvalidItem, stated.key)
		})
	}
}

func TestAConfigurationThatIsNotExactlyOneObjectOfItemsIsRefusedAsMalformed(t *testing.T) {
	complete := encoded(t, completeItems(t))
	for form, content := range map[string]string{
		"an inline secret":  strings.Replace(complete, "{", `{"webhookSecret":"s3cret",`, 1),
		"a key in capitals": strings.Replace(complete, `"appId"`, `"APPID"`, 1),
		"a key given twice": strings.Replace(complete, "{", `{"appId":1,`, 1),
		"a second document": complete + `{}`,
		"trailing text":     complete + ` trailing`,
		"an array":          `[` + complete + `]`,
	} {
		t.Run(form, func(t *testing.T) {
			_, err := configuration.Read(writtenFile(t, "controlplane.json", content))

			if !errors.Is(err, configuration.ErrMalformedConfiguration) {
				t.Fatalf("refusal is not a malformed configuration: %v", err)
			}
		})
	}
}

func TestAConfigurationFileAtTheBoundIsReadAndOneByteMoreIsRefused(t *testing.T) {
	complete := encoded(t, completeItems(t))
	atBound := complete + strings.Repeat(" ", configuration.MaxFileBytes-len(complete))

	if _, err := configuration.Read(writtenFile(t, "controlplane.json", atBound)); err != nil {
		t.Fatalf("a configuration of exactly %d bytes was refused: %v", len(atBound), err)
	}
	_, err := configuration.Read(writtenFile(t, "controlplane.json", atBound+" "))
	if !errors.Is(err, configuration.ErrOversizedFile) {
		t.Fatalf("a configuration one byte over the bound was not refused as oversized: %v", err)
	}
}

func TestAConfigurationFileThatCannotBeOpenedIsRefused(t *testing.T) {
	_, err := configuration.Read(filepath.Join(t.TempDir(), "absent.json"))

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("an absent configuration file was not refused as absent: %v", err)
	}
}

func TestNoFormattingVerbPrintsASecretTheConfigurationHolds(t *testing.T) {
	configured := readConfiguration(t, completeItems(t))
	held := map[string]func(verb string) string{
		"the configuration":              func(verb string) string { return fmt.Sprintf(verb, configured) },
		"a pointer to the configuration": func(verb string) string { return fmt.Sprintf(verb, &configured) },
		"the App key":                    func(verb string) string { return fmt.Sprintf(verb, configured.AppKey()) },
		"the webhook secret":             func(verb string) string { return fmt.Sprintf(verb, configured.WebhookSecret()) },
	}

	for name, printedUnder := range held {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%d", "%x", "%X", "%q", "%p", "%+d", "%#x"} {
			printed := printedUnder(verb)
			for _, form := range printedFormsOf(statedWebhookSecret, statedAppKeyBody) {
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

type statedItems map[string]json.RawMessage

func completeItems(t *testing.T) statedItems {
	t.Helper()
	return statedItems{
		listenAddressKey:     quoted(t, statedListenAddress),
		appIDKey:             number(statedAppID),
		appKeyFileKey:        quoted(t, writtenFile(t, "app-key.pem", statedAppKey)),
		webhookSecretFileKey: quoted(t, writtenFile(t, "webhook-secret", statedWebhookSecret)),
		installationIDsKey:   listOf(number(statedInstallationID)),
		familiesKey:          listOf(quoted(t, string(domain.PolicyFamilyStructural))),
	}
}

func quoted(t *testing.T, text string) json.RawMessage {
	t.Helper()
	content, err := json.Marshal(text)
	if err != nil {
		t.Fatalf("encode %q: %v", text, err)
	}
	return content
}

func listOf(values ...[]byte) json.RawMessage {
	return json.RawMessage("[" + string(bytes.Join(values, []byte(","))) + "]")
}

func number(stated int64) json.RawMessage {
	return json.RawMessage(strconv.FormatInt(stated, 10))
}

func readConfiguration(t *testing.T, stated statedItems) configuration.Configuration {
	t.Helper()
	configured, err := configuration.Read(writtenFile(t, "controlplane.json", encoded(t, stated)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return configured
}

func refusalOf(t *testing.T, stated statedItems) error {
	t.Helper()
	_, err := configuration.Read(writtenFile(t, "controlplane.json", encoded(t, stated)))
	if err == nil {
		t.Fatal("the configuration was read, want a refusal")
	}
	return err
}

func assertOnlyRefusal(t *testing.T, err error, refusal error, key string) {
	t.Helper()
	parts := refusalsIn(err)
	if len(parts) != 1 {
		t.Fatalf("want one refusal, got %d: %v", len(parts), err)
	}
	if !errors.Is(parts[0], refusal) || !strings.Contains(parts[0].Error(), key) {
		t.Fatalf("refusal is not %v naming %s: %v", refusal, key, parts[0])
	}
}

func refusalsIn(err error) []error {
	joined, isJoined := err.(interface{ Unwrap() []error })
	if !isJoined {
		return []error{err}
	}
	return joined.Unwrap()
}

func encoded(t *testing.T, stated statedItems) string {
	t.Helper()
	content, err := json.Marshal(stated)
	if err != nil {
		t.Fatalf("encode the configuration: %v", err)
	}
	return string(content)
}

func writtenFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
