// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const checkRunWait = 20 * time.Second

var openedPullRequest = `{"action":"opened","number":7,"repository":{"full_name":"JoeNutt/open-ci-control-plane"},` +
	`"installation":{"id":` + strconv.Itoa(installationID) + `}}`

func TestASignedOpenedPullRequestIsAcceptedAndItsCheckRunReachesTheForge(t *testing.T) {
	recording := &recordingForge{t: t, published: make(chan struct{}, 1)}
	stated, stop := servedAgainst(t, recording)
	defer stop()

	answered := deliveredTo(t, addressIn(t, stated), openedPullRequest, signatureOf(openedPullRequest))

	if answered != http.StatusAccepted {
		t.Errorf("a signed opened pull request was answered %d, want %d", answered, http.StatusAccepted)
	}
	select {
	case <-recording.published:
	case <-time.After(checkRunWait):
		t.Error("no check run reached the forge")
	}
}

func TestAWronglySignedPullRequestIsRefusedAndNoCheckRunIsPosted(t *testing.T) {
	recording := &recordingForge{t: t, published: make(chan struct{}, 1)}
	stated, stop := servedAgainst(t, recording)

	answered := deliveredTo(t, addressIn(t, stated), openedPullRequest, signatureOf(openedPullRequest+" "))
	stop()

	if answered != http.StatusUnauthorized {
		t.Errorf("a wrongly signed delivery was answered %d, want %d", answered, http.StatusUnauthorized)
	}
	select {
	case <-recording.published:
		t.Error("a check run was posted for a wrongly signed delivery")
	default:
	}
}

func servedAgainst(t *testing.T, recording *recordingForge) (map[string]json.RawMessage, func()) {
	t.Helper()
	server := recordingServer(t, recording)
	trusted := x509.NewCertPool()
	trusted.AddCert(server.Certificate())
	stated := completeConfiguration(t)
	serving, stop := context.WithCancel(context.Background())
	exited := make(chan int)
	var logged bytes.Buffer
	go func() {
		exited <- runAgainst(serving, []string{configurationFile(t, stated)}, &logged,
			func() (forgeEndpoint, error) { return forgeEndpoint{baseURL: server.URL, trusted: trusted}, nil })
	}()
	return stated, func() {
		stop()
		if exit := <-exited; exit != exitServed {
			t.Errorf("exit is %d, want %d: %s", exit, exitServed, logged.String())
		}
	}
}

func deliveredTo(t *testing.T, listenAddress, payload, signature string) int {
	t.Helper()
	for range 200 {
		delivery, err := http.NewRequest(http.MethodPost, "http://"+listenAddress+"/v1/forge/github/events",
			strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		delivery.Header.Set("X-GitHub-Event", "pull_request")
		delivery.Header.Set("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
		delivery.Header.Set("X-Hub-Signature-256", signature)
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
