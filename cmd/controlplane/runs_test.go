// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"slices"
	"strconv"
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
