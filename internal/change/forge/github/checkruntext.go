// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var checkRunTextErrUnidentified = errors.New("check run carries no decision identifier")

const checkRunTextSummaryLayout = "%s\n\nDecision `%s`"

type checkRunTextOutput struct {
	title   string
	summary string
	text    string
}

func checkRunTextFor(run domain.CheckRun, capacity int) (checkRunTextOutput, error) {
	if run.DecisionID == (domain.DecisionID{}) {
		return checkRunTextOutput{}, checkRunTextErrUnidentified
	}
	text := ""
	for _, violation := range run.Violations {
		text += "**Rule** ` " + violation.RuleID + " ` — " + violation.Marking + "\n\n" +
			"Message: ` " + violation.Message + " `\n\n" +
			"- Owning team: ` " + violation.OwningTeam + " `\n\n"
	}
	return checkRunTextOutput{
		title:   run.Title,
		summary: fmt.Sprintf(checkRunTextSummaryLayout, run.Headline, run.DecisionID),
		text:    text,
	}, nil
}
