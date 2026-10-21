// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAConfigurationMissingARequiredItemExitsNonZeroNamingItAndServesNothing(t *testing.T) {
	for _, item := range []string{"appKeyFile", "webhookSecretFile", "installationIds"} {
		t.Run(item, func(t *testing.T) {
			stated := completeConfiguration(t)
			delete(stated, item)
			var logged bytes.Buffer

			exit := run(context.Background(), []string{configurationFile(t, stated)}, &logged)

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

			exit := run(context.Background(), arguments, &logged)

			if exit == 0 {
				t.Errorf("exit is 0 for %s", form)
			}
			if !strings.Contains(logged.String(), "one configuration file path") {
				t.Errorf("the refusal does not say what is expected: %q", logged.String())
			}
		})
	}
}

func TestACompleteConfigurationServesTheEventsRouteAndLogsWhatItServes(t *testing.T) {
	serving, stop := context.WithCancel(context.Background())
	var logged bytes.Buffer
	exited := make(chan int)
	go func() {
		exited <- run(serving, []string{configurationFile(t, completeConfiguration(t))}, &logged)
	}()

	answered := firstAnswerFromTheListener(t)
	stop()
	exit := <-exited

	if answered != http.StatusUnauthorized {
		t.Errorf("an unsigned delivery was answered %d, want %d", answered, http.StatusUnauthorized)
	}
	if exit != 0 {
		t.Errorf("exit is %d after a clean stop, want 0", exit)
	}
	for _, named := range []string{`"structural"`, "52311907"} {
		if !strings.Contains(logged.String(), named) {
			t.Errorf("the start-up log does not name %s: %q", named, logged.String())
		}
	}
}

func firstAnswerFromTheListener(t *testing.T) int {
	t.Helper()
	for attempt := 0; attempt < 200; attempt++ {
		delivery, err := http.NewRequest(http.MethodPost, "http://"+listenAddress+"/v1/forge/github/events",
			strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		delivery.Header.Set("X-GitHub-Event", "pull_request")
		delivery.Header.Set("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
		answered, err := http.DefaultClient.Do(delivery)
		if err == nil {
			defer answered.Body.Close()
			return answered.StatusCode
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the listener never answered")
	return 0
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
