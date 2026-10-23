// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"slices"
	"strconv"
	"testing"
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
