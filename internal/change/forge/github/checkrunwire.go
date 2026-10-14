// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const statusCompleted = "completed"

func creationOf(locator domain.PullRequestLocator, head commitSHA, run domain.CheckRun) exchange {
	return exchange{
		method:   http.MethodPost,
		endpoint: checkRunsEndpoint(locator),
		sent: newCheckRun{
			Name:            run.Name,
			HeadSHA:         string(head),
			checkRunVerdict: verdictOf(run),
		},
		succeeded: http.StatusCreated,
	}
}

func updateOf(locator domain.PullRequestLocator, id checkRunID, run domain.CheckRun) exchange {
	return exchange{
		method:    http.MethodPatch,
		endpoint:  fmt.Sprintf("%s/%d", checkRunsEndpoint(locator), id),
		sent:      verdictOf(run),
		succeeded: http.StatusOK,
	}
}

func checkRunsEndpoint(locator domain.PullRequestLocator) string {
	return fmt.Sprintf("/repos/%s/%s/check-runs", locator.Owner(), locator.Name())
}

func checkRunsOnCommitEndpoint(
	locator domain.PullRequestLocator, head commitSHA, name string,
) string {
	query := url.Values{"check_name": {name}}
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?%s",
		locator.Owner(), locator.Name(), head, query.Encode())
}

type checkRunID int64

type checkRunListing struct {
	TotalCount int              `json:"total_count"`
	CheckRuns  []listedCheckRun `json:"check_runs"`
}

type listedCheckRun struct {
	ID checkRunID `json:"id"`
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
