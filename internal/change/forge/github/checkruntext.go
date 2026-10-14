// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var checkRunTextErrUnidentified = errors.New("check run carries no decision identifier")

const (
	checkRunTextSummaryLayout   = "%s\n\nDecision `%s`"
	checkRunTextViolationLayout = "**Rule** %s — %s\n\nMessage: %s\n\n- Owning team: %s\n- Documentation: %s\n\n"
)

type checkRunTextOutput struct {
	title   string
	summary string
	text    string
}

func checkRunTextFor(run domain.CheckRun, capacity int) (checkRunTextOutput, error) {
	if run.DecisionID == (domain.DecisionID{}) {
		return checkRunTextOutput{}, checkRunTextErrUnidentified
	}
	text := checkRunTextViolationsOf(run.Violations)
	if run.ViolationCount > len(run.Violations) {
		text += fmt.Sprintf("and %d more — %d violations in total. This list is truncated.\n",
			run.ViolationCount-len(run.Violations), run.ViolationCount)
	}
	return checkRunTextOutput{
		title:   run.Title,
		summary: fmt.Sprintf(checkRunTextSummaryLayout, run.Headline, run.DecisionID),
		text:    text,
	}, nil
}

func checkRunTextViolationsOf(violations []domain.CheckRunViolation) string {
	var text strings.Builder
	for _, violation := range violations {
		fmt.Fprintf(&text, checkRunTextViolationLayout,
			markdownCodeSpanOf(violation.RuleID),
			violation.Marking,
			markdownCodeSpanOf(violation.Message),
			markdownCodeSpanOf(violation.OwningTeam),
			checkRunTextDocumentationOf(violation.DocsURL))
	}
	return text.String()
}
