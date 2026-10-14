// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestARepublishedCheckRunReplacesTheOneWithItsNameOnTheSameCommit(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	runs := &memory.CheckRuns{}
	first := domain.CheckRun{Name: "policy/security", Conclusion: domain.ConclusionFailure}
	second := domain.CheckRun{Name: "policy/security", Conclusion: domain.ConclusionSuccess}
	other := domain.CheckRun{Name: "policy/quality", Conclusion: domain.ConclusionNeutral}

	for _, run := range []domain.CheckRun{first, other, second} {
		if err := runs.PublishCheckRun(context.Background(), locator, "9f1cabc", run); err != nil {
			t.Fatalf("publish %s: %v", run.Name, err)
		}
	}

	published := runs.PublishedOn(locator, "9f1cabc")
	if len(published) != 2 {
		t.Fatalf("got %d check runs on the commit, want 2: %+v", len(published), published)
	}
	if !reflect.DeepEqual(published["policy/security"], second) {
		t.Errorf("policy/security is %+v, want the republished %+v",
			published["policy/security"], second)
	}
	if !reflect.DeepEqual(published["policy/quality"], other) {
		t.Errorf("policy/quality is %+v, want %+v", published["policy/quality"], other)
	}
	if elsewhere := runs.PublishedOn(locator, "0000000"); len(elsewhere) != 0 {
		t.Errorf("another commit shows check runs %+v, want none", elsewhere)
	}
}
