// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestATestsCheckRunIsPublishedUnderTheRepositorysOwnAllowlistedInstallation(t *testing.T) {
	recording := &recordingForge{t: t, installedAs: installationID}
	publisher, err := recordingAppForge(t, recording).repositoryCheckRunsFor([]int64{installationID - 1, installationID})
	if err != nil {
		t.Fatalf("build the publisher: %v", err)
	}

	publishCheckRun(t, publisher)

	if want := []string{strconv.Itoa(installationID)}; !slices.Equal(recording.exchanged, want) {
		t.Errorf("tokens were minted for installations %v, want %v", recording.exchanged, want)
	}
}

func TestARepositoryInstalledOffTheAllowlistIsRefusedBeforeAnyTokenIsMinted(t *testing.T) {
	recording := &recordingForge{t: t, installedAs: installationID + 1}
	publisher, err := recordingAppForge(t, recording).repositoryCheckRunsFor([]int64{installationID})
	if err != nil {
		t.Fatalf("build the publisher: %v", err)
	}

	err = publisher.PublishCheckRun(context.Background(), recordedPullRequest(t), recordedHead(t), domain.CheckRun{Name: domain.TestsCheckRunName})

	if !errors.Is(err, errInstallationNotAllowlisted) {
		t.Errorf("publishing under an installation off the allowlist gave %v, want %v", err, errInstallationNotAllowlisted)
	}
	if len(recording.exchanged) != 0 || len(recording.presented) != 0 {
		t.Errorf("minted %v and presented %v for an installation off the allowlist", recording.exchanged, recording.presented)
	}
}

func recordedHead(t *testing.T) domain.CommitSHA {
	t.Helper()

	head, err := domain.ParseCommitSHA("b0b0000000000000000000000000000000000b0b")
	if err != nil {
		t.Fatal(err)
	}
	return head
}

func TestTheBinaryServesTheRunsRouteAndRefusesAReportWithNoToken(t *testing.T) {
	serving, stop := context.WithCancel(context.Background())
	var logged bytes.Buffer
	exited := make(chan int)
	stated := completeConfiguration(t)
	go func() {
		exited <- run(serving, []string{configurationFile(t, stated)}, &logged)
	}()
	firstAnswerFrom(t, addressIn(t, stated), "")

	report, err := http.NewRequest(http.MethodPut,
		"http://"+addressIn(t, stated)+"/v1/repositories/JoeNutt/open-ci-control-plane/runs/b0b0000000000000000000000000000000000b0b",
		strings.NewReader(`{"decisionId":"dec_0123456789abcdef0123456789abcdef","tests":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	answered, err := http.DefaultClient.Do(report)
	stop()
	<-exited
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if err := answered.Body.Close(); err != nil {
		t.Errorf("close the answer: %v", err)
	}

	if answered.StatusCode != http.StatusUnauthorized {
		t.Errorf("a report with no token was answered %d, want %d: %s", answered.StatusCode, http.StatusUnauthorized, logged.String())
	}
}
