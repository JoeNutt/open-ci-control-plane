// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const statusCompleted = "completed"

type publication struct {
	locator domain.PullRequestLocator
	head    commitSHA
	run     domain.CheckRun
}

func (pending publication) exchangeGiven(listed checkRunListing) (exchange, error) {
	if listed.TotalCount > 1 {
		return exchange{}, fmt.Errorf("%w: %d named %q on %s",
			ErrAmbiguousCheckRun, listed.TotalCount, pending.run.Name, pending.head)
	}
	if len(listed.CheckRuns) == 0 {
		return pending.creation(), nil
	}
	return pending.updateOf(listed.CheckRuns[0].ID), nil
}

func (pending publication) creation() exchange {
	return exchange{
		method:   http.MethodPost,
		endpoint: pending.checkRunsEndpoint(),
		sent: newCheckRun{
			Name:            pending.run.Name,
			HeadSHA:         string(pending.head),
			checkRunVerdict: verdictOf(pending.run),
		},
		succeeded: http.StatusCreated,
	}
}

func (pending publication) updateOf(id checkRunID) exchange {
	return exchange{
		method:    http.MethodPatch,
		endpoint:  fmt.Sprintf("%s/%d", pending.checkRunsEndpoint(), id),
		sent:      verdictOf(pending.run),
		succeeded: http.StatusOK,
	}
}

func (pending publication) checkRunsEndpoint() string {
	return fmt.Sprintf("/repos/%s/%s/check-runs",
		pending.locator.Owner(), pending.locator.Name())
}

func (pending publication) listingEndpoint() string {
	query := url.Values{"check_name": {pending.run.Name}}
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?%s",
		pending.locator.Owner(), pending.locator.Name(), pending.head, query.Encode())
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
