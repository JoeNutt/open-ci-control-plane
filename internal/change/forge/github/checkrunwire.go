// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const statusCompleted = "completed"

func checkRunsEndpoint(locator domain.PullRequestLocator) string {
	return fmt.Sprintf("/repos/%s/%s/check-runs", locator.Owner(), locator.Name())
}

func checkRunsOnCommitEndpoint(
	locator domain.PullRequestLocator, headSHA, name string,
) string {
	query := url.Values{"check_name": {name}}
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?%s",
		locator.Owner(), locator.Name(), headSHA, query.Encode())
}

type checkRunListing struct {
	TotalCount int `json:"total_count"`
}

type checkRunOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text"`
}

type checkRunVerdict struct {
	Status     string         `json:"status"`
	Conclusion string         `json:"conclusion"`
	Output     checkRunOutput `json:"output"`
}

type newCheckRun struct {
	Name    string `json:"name"`
	HeadSHA string `json:"head_sha"`
	checkRunVerdict
}

func verdictOf(run domain.CheckRun) checkRunVerdict {
	return checkRunVerdict{
		Status:     statusCompleted,
		Conclusion: string(run.Conclusion),
		Output:     checkRunOutput{Title: run.Title, Summary: run.Summary, Text: run.Text},
	}
}
