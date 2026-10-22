// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
)

func TestAConfigurationMissingOrInvalidInARequiredItemExitsNonZeroNamingItAndServesNothing(t *testing.T) {
	refused := map[string]struct {
		item  string
		state func(t *testing.T, stated map[string]json.RawMessage)
	}{
		"no App key":        {"appKeyFile", without("appKeyFile")},
		"no webhook secret": {"webhookSecretFile", without("webhookSecretFile")},
		"no allowlist":      {"installationIds", without("installationIds")},
		"an App key that is not an RSA key": {"appKeyFile", func(t *testing.T, stated map[string]json.RawMessage) {
			stated["appKeyFile"] = quoted(t, writtenFile(t, t.TempDir(), "app-key.pem", unparseableAppKey))
		}},
		"an unreadable App key": {"appKeyFile", stating("appKeyFile", `"/nonexistent/app-key.pem"`)},
		"an empty webhook secret": {"webhookSecretFile", func(t *testing.T, stated map[string]json.RawMessage) {
			stated["webhookSecretFile"] = quoted(t, writtenFile(t, t.TempDir(), "empty", "\n"))
		}},
		"an empty allowlist": {"installationIds", stating("installationIds", `[]`)},
	}
	for form, configured := range refused {
		t.Run(form, func(t *testing.T) {
			stated := completeConfiguration(t)
			configured.state(t, stated)
			var logged bytes.Buffer

			exit := run(context.Background(), []string{configurationFile(t, stated)}, &logged)

			if exit == 0 {
				t.Errorf("exit is 0 for %s", form)
			}
			if !strings.Contains(logged.String(), configured.item) {
				t.Errorf("the refusal does not name %s: %q", configured.item, logged.String())
			}
			assertNothingServesOn(t, addressIn(t, stated))
		})
	}
}

func without(item string) func(*testing.T, map[string]json.RawMessage) {
	return func(_ *testing.T, stated map[string]json.RawMessage) { delete(stated, item) }
}

func stating(item, value string) func(*testing.T, map[string]json.RawMessage) {
	return func(_ *testing.T, stated map[string]json.RawMessage) { stated[item] = json.RawMessage(value) }
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
	answered, exit, logged := servedOnce(t, "")

	if answered != http.StatusUnauthorized {
		t.Errorf("an unsigned delivery was answered %d, want %d", answered, http.StatusUnauthorized)
	}
	if exit != 0 {
		t.Errorf("exit is %d after a clean stop, want 0", exit)
	}
	for _, named := range []string{`"structural"`, strconv.Itoa(installationID)} {
		if !strings.Contains(logged.String(), named) {
			t.Errorf("the start-up log does not name %s: %q", named, logged.String())
		}
	}
}

func TestTheBinaryStartsEachFamilysEngineFromThePolicyBundleItServes(t *testing.T) {
	_, exit, logged := servedOnce(t, "")

	if exit != 0 {
		t.Errorf("exit is %d after a clean stop, want 0", exit)
	}
	for _, named := range []string{enginesStarted, "JoeNutt/open-ci-policy"} {
		if !strings.Contains(logged.String(), named) {
			t.Errorf("the start-up log does not name %s: %q", named, logged.String())
		}
	}
}

func TestNoStartUpOrServingLogCarriesTheWebhookSecretOrTheAppKey(t *testing.T) {
	_, _, logged := servedOnce(t, "")

	keyLines := strings.Split(rsaAppKeyPEM(t), "\n")
	for name, secret := range map[string]string{"the webhook secret": webhookSecret, "the App key": keyLines[1]} {
		if strings.Contains(logged.String(), secret) {
			t.Errorf("a log line carries %s: %q", name, logged.String())
		}
	}
}

var _ httpapi.ChangeGate = application.GateChange{}

func servedOnce(t *testing.T, signedPayload string) (int, int, *bytes.Buffer) {
	t.Helper()
	serving, stop := context.WithCancel(context.Background())
	var logged bytes.Buffer
	exited := make(chan int)
	stated := completeConfiguration(t)
	go func() {
		exited <- run(serving, []string{configurationFile(t, stated)}, &logged)
	}()
	answered := firstAnswerFrom(t, addressIn(t, stated), signedPayload)
	stop()
	return answered, <-exited, &logged
}

func firstAnswerFrom(t *testing.T, listenAddress, signedPayload string) int {
	t.Helper()
	for attempt := 0; attempt < 200; attempt++ {
		delivery, err := http.NewRequest(http.MethodPost, "http://"+listenAddress+"/v1/forge/github/events",
			strings.NewReader(signedPayload))
		if err != nil {
			t.Fatal(err)
		}
		delivery.Header.Set("X-GitHub-Event", "pull_request")
		delivery.Header.Set("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
		if signedPayload != "" {
			delivery.Header.Set("X-Hub-Signature-256", signatureOf(signedPayload))
		}
		answered, err := http.DefaultClient.Do(delivery)
		if err == nil {
			if err := answered.Body.Close(); err != nil {
				t.Errorf("close the answer: %v", err)
			}
			return answered.StatusCode
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the listener never answered")
	return 0
}

const (
	listenAddressItem   = "listenAddress"
	webhookSecret       = "the App's webhook secret"
	appKeyBody          = "c3RhdGVkIGtleQ"
	appID               = 1204331
	installationID      = 52311907
	classificationRules = "test: [\"**/*_test.go\"]\n"
)

const unparseableAppKey = "-----BEGIN RSA PRIVATE KEY-----\n" + appKeyBody + "\n-----END RSA PRIVATE KEY-----\n"

func assertNothingServesOn(t *testing.T, listenAddress string) {
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
	appKey := writtenFile(t, directory, "app-key.pem", rsaAppKeyPEM(t))
	secret := writtenFile(t, directory, "webhook-secret", webhookSecret+"\n")
	return map[string]json.RawMessage{
		listenAddressItem:         quoted(t, unusedLoopbackAddress(t)),
		"appId":                   json.RawMessage(strconv.Itoa(appID)),
		"appKeyFile":              quoted(t, appKey),
		"webhookSecretFile":       quoted(t, secret),
		"installationIds":         json.RawMessage("[" + strconv.Itoa(installationID) + "]"),
		"families":                json.RawMessage(`["structural"]`),
		"policyRepository":        quoted(t, "JoeNutt/open-ci-policy"),
		"policyInstallationId":    json.RawMessage(strconv.Itoa(installationID)),
		"classificationRulesFile": quoted(t, writtenFile(t, directory, "rules.yaml", classificationRules)),
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

func signatureOf(payload string) string {
	signing := hmac.New(sha256.New, []byte(webhookSecret))
	signing.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(signing.Sum(nil))
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

func unusedLoopbackAddress(t *testing.T) string {
	t.Helper()
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find an unused loopback port: %v", err)
	}
	address := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatalf("release the unused loopback port: %v", err)
	}
	return address
}

func addressIn(t *testing.T, stated map[string]json.RawMessage) string {
	t.Helper()
	var address string
	if err := json.Unmarshal(stated[listenAddressItem], &address); err != nil {
		t.Fatalf("read the stated listen address: %v", err)
	}
	return address
}
