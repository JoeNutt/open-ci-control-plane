// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const statusCompleted = "completed"

var ErrInconsistentListing = errors.New("the forge's check run count disagrees with the runs it listed")

type checkRunID int64

type checkRunListing struct {
	TotalCount int              `json:"total_count"`
	CheckRuns  []listedCheckRun `json:"check_runs"`
}

func (listed checkRunListing) consistentRuns() ([]listedCheckRun, error) {
	if listed.TotalCount != len(listed.CheckRuns) {
		return nil, fmt.Errorf("%w: a count of %d beside %d runs",
			ErrInconsistentListing, listed.TotalCount, len(listed.CheckRuns))
	}
	return listed.CheckRuns, nil
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

func parseVerdict(run domain.CheckRun) (checkRunVerdict, error) {
	conclusion, err := domain.ParseConclusion(string(run.Conclusion))
	if err != nil {
		return checkRunVerdict{}, err
	}
	return checkRunVerdict{
		Status:     statusCompleted,
		Conclusion: string(conclusion),
		Output:     checkRunOutput{Title: run.Title, Summary: run.Summary, Text: run.Text},
	}, nil
}
