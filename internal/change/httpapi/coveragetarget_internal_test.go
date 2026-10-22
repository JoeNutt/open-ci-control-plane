// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	targetOwner = "JoeNutt"
	targetRepo  = "open-ci-control-plane"
	targetSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestACoverageTargetIsTheRepositoryAndSuiteItsPathNames(t *testing.T) {
	target, refused := coverageTargetOf(targetOwner, targetRepo, targetSuite)

	if refused != nil {
		t.Fatalf("refused %v", refused)
	}
	if target.repository.String() != targetOwner+"/"+targetRepo || target.suite.String() != targetSuite {
		t.Errorf("target is %s %s, want %s/%s %s", target.repository, target.suite, targetOwner, targetRepo, targetSuite)
	}
}

func TestACoverageTargetIsRefusedNamingTheSegmentThatIsNotOneC12Admits(t *testing.T) {
	for form, stated := range map[string]struct{ owner, repo, suite, segment string }{
		"an owner with an underscore":   {"Joe_Nutt", targetRepo, targetSuite, repositorySegments},
		"an owner opening with a dash":  {"-JoeNutt", targetRepo, targetSuite, repositorySegments},
		"a repo with a space":           {targetOwner, "open ci", targetSuite, repositorySegments},
		"a repo of dots":                {targetOwner, "..", targetSuite, repositorySegments},
		"a suite with a plus":           {targetOwner, targetRepo, "github.com/JoeNutt/open+ci", suiteSegment},
		"a suite with a dot-dot":        {targetOwner, targetRepo, "github.com/JoeNutt/../domain", suiteSegment},
		"a suite with an empty segment": {targetOwner, targetRepo, "github.com//domain", suiteSegment},
		"a suite over its bound": {
			targetOwner, targetRepo, "github.com/" + strings.Repeat("s", domain.MaxGoImportPathBytes), suiteSegment,
		},
	} {
		t.Run(form, func(t *testing.T) {
			_, refused := coverageTargetOf(stated.owner, stated.repo, stated.suite)

			if refused == nil || refused.status() != http.StatusBadRequest || refused.(fieldProblem).Field != stated.segment {
				t.Errorf("refused %#v, want a 400 naming %s", refused, stated.segment)
			}
		})
	}
}
