// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var (
	errCheckRunTextUnidentified = errors.New("check run carries no decision identifier")
	errCheckRunTextOverCapacity = errors.New("check run output cannot fit its capacity")
	errCheckRunTextUndercounted = errors.New("check run counts fewer violations than it carries")
	errCheckRunTextOverlisted   = errors.New("check run lists more violations than the domain ever carries")
)

const (
	checkRunTextSummaryLayout    = "%s\n\nDecision `%s`"
	checkRunTextTruncationLayout = "and %d more — %d %s in total. This list is truncated.\n"
	checkRunTextNoneListedLayout = "No violation could be listed here — %d %s in total. This list is truncated.\n"
	checkRunTextViolationLayout  = "**Rule** %s — %s\n\nMessage: %s\n\n- Owning team: %s\n- Documentation: %s\n\n"
)

type checkRunTextOutput struct {
	title   string
	summary string
	text    string
}

func checkRunTextFor(run domain.CheckRun, capacity int) (checkRunTextOutput, error) {
	if err := checkRunTextRefusalOf(run); err != nil {
		return checkRunTextOutput{}, err
	}
	blocks := checkRunTextBlocksThatFit(run, capacity)
	output := checkRunTextOutput{
		title:   run.Title,
		summary: fmt.Sprintf(checkRunTextSummaryLayout, run.Headline, run.DecisionID),
		text:    strings.Join(blocks, "") + checkRunTextTruncationOf(run.ViolationCount-len(blocks), run.ViolationCount),
	}
	if len(output.summary) > capacity || len(output.text) > capacity {
		return checkRunTextOutput{}, errCheckRunTextOverCapacity
	}
	return output, nil
}

func checkRunTextRefusalOf(run domain.CheckRun) error {
	switch {
	case run.DecisionID == (domain.DecisionID{}):
		return errCheckRunTextUnidentified
	case run.ViolationCount < len(run.Violations):
		return errCheckRunTextUndercounted
	case len(run.Violations) > domain.ListedViolationLimit:
		return errCheckRunTextOverlisted
	default:
		return nil
	}
}

func checkRunTextBlocksThatFit(run domain.CheckRun, capacity int) []string {
	var blocks []string
	used := 0
	for _, violation := range run.Violations {
		least := 0
		for _, field := range []string{violation.RuleID, violation.Message, violation.OwningTeam, violation.DocsURL, violation.Marking} {
			least += utf8.RuneCountInString(field) - strings.Count(field, "\r\n")
		}
		if used+least > capacity {
			return blocks
		}
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
	if leftOut == total {
		return fmt.Sprintf(checkRunTextNoneListedLayout, total, checkRunTextViolationNounFor(total))
	}
	return fmt.Sprintf(checkRunTextTruncationLayout, leftOut, total, checkRunTextViolationNounFor(total))
}

func checkRunTextViolationNounFor(total int) string {
	if total == 1 {
		return "violation"
	}
	return "violations"
}

func checkRunTextBlockOf(violation domain.CheckRunViolation) string {
	return fmt.Sprintf(checkRunTextViolationLayout,
		markdownCodeSpanOf(violation.RuleID),
		violation.Marking,
		markdownCodeSpanOf(violation.Message),
		markdownCodeSpanOf(violation.OwningTeam),
		checkRunTextDocumentationOf(violation.DocsURL))
}
