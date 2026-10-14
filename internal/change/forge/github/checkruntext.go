// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var (
	checkRunTextErrUnidentified = errors.New("check run carries no decision identifier")
	checkRunTextErrOverCapacity = errors.New("check run output cannot fit its capacity")
)

const (
	checkRunTextSummaryLayout    = "%s\n\nDecision `%s`"
	checkRunTextTruncationLayout = "and %d more — %d violations in total. This list is truncated.\n"
	checkRunTextViolationLayout  = "**Rule** %s — %s\n\nMessage: %s\n\n- Owning team: %s\n- Documentation: %s\n\n"
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
	blocks := checkRunTextBlocksThatFit(run, capacity)
	output := checkRunTextOutput{
		title:   run.Title,
		summary: fmt.Sprintf(checkRunTextSummaryLayout, run.Headline, run.DecisionID),
		text:    strings.Join(blocks, "") + checkRunTextTruncationOf(run.ViolationCount-len(blocks), run.ViolationCount),
	}
	if len(output.summary) > capacity || len(output.text) > capacity {
		return checkRunTextOutput{}, checkRunTextErrOverCapacity
	}
	return output, nil
}

func checkRunTextBlocksThatFit(run domain.CheckRun, capacity int) []string {
	var blocks []string
	used := 0
	for _, violation := range run.Violations {
		block := checkRunTextBlockOf(violation)
		closing := checkRunTextTruncationOf(run.ViolationCount-len(blocks)-1, run.ViolationCount)
		if used+len(block)+len(closing) > capacity {
			return blocks
		}
		blocks = append(blocks, block)
		used += len(block)
	}
	return blocks
}

func checkRunTextTruncationOf(leftOut, total int) string {
	if leftOut <= 0 {
		return ""
	}
	return fmt.Sprintf(checkRunTextTruncationLayout, leftOut, total)
}

func checkRunTextBlockOf(violation domain.CheckRunViolation) string {
	return fmt.Sprintf(checkRunTextViolationLayout,
		markdownCodeSpanOf(violation.RuleID),
		violation.Marking,
		markdownCodeSpanOf(violation.Message),
		markdownCodeSpanOf(violation.OwningTeam),
		checkRunTextDocumentationOf(violation.DocsURL))
}
