// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAConfigurationMissingARequiredItemExitsNonZeroNamingItAndServesNothing(t *testing.T) {
	for _, item := range []string{"appKeyFile", "webhookSecretFile", "installationIds"} {
		t.Run(item, func(t *testing.T) {
			stated := completeConfiguration(t)
			delete(stated, item)
			var logged bytes.Buffer

			exit := run([]string{configurationFile(t, stated)}, &logged)

			if exit == 0 {
				t.Errorf("exit is 0 for a configuration with no %s", item)
			}
			if !strings.Contains(logged.String(), item) {
				t.Errorf("the refusal does not name %s: %q", item, logged.String())
			}
			assertNothingServes(t)
		})
	}
}

func TestAnythingButOneConfigurationPathExitsNonZeroSayingSo(t *testing.T) {
	for form, arguments := range map[string][]string{
		"no argument":   nil,
		"two arguments": {"one.json", "two.json"},
	} {
		t.Run(form, func(t *testing.T) {
			var logged bytes.Buffer

			exit := run(arguments, &logged)

			if exit == 0 {
				t.Errorf("exit is 0 for %s", form)
			}
			if !strings.Contains(logged.String(), "one configuration file path") {
				t.Errorf("the refusal does not say what is expected: %q", logged.String())
			}
		})
	}
}

const listenAddress = "127.0.0.1:48121"

func assertNothingServes(t *testing.T) {
	t.Helper()
	listening, err := net.Dial("tcp", listenAddress)
	if err != nil {
		return
	}
	t.Errorf("something is serving on %s", listenAddress)
	if err := listening.Close(); err != nil {
		t.Errorf("close the probe: %v", err)
	}
}

func completeConfiguration(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	directory := t.TempDir()
	appKey := writtenFile(t, directory, "app-key.pem",
		"-----BEGIN RSA PRIVATE KEY-----\nc3RhdGVkIGtleQ\n-----END RSA PRIVATE KEY-----\n")
	secret := writtenFile(t, directory, "webhook-secret", "the App's webhook secret\n")
	return map[string]json.RawMessage{
		"listenAddress":     quoted(t, listenAddress),
		"appId":             json.RawMessage(`1204331`),
		"appKeyFile":        quoted(t, appKey),
		"webhookSecretFile": quoted(t, secret),
		"installationIds":   json.RawMessage(`[52311907]`),
		"families":          json.RawMessage(`["structural"]`),
	}
}

func configurationFile(t *testing.T, stated map[string]json.RawMessage) string {
	t.Helper()
	encoded, err := json.Marshal(stated)
	if err != nil {
		t.Fatal(err)
	}
	return writtenFile(t, t.TempDir(), "configuration.json", string(encoded))
}

func writtenFile(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func quoted(t *testing.T, text string) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
