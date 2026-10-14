// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

const (
	statusCompleted = "completed"
	filterLatest    = "latest"
)

// GitHub serves at most 100 check runs per page of a listing:
// https://docs.github.com/en/rest/checks/runs
const CheckRunsPerPage = 100

var ErrMalformedListing = errors.New("the forge's check run listing does not state its count and runs")

var ErrInconsistentListing = errors.New("the forge's check run count disagrees with the runs it listed")

var ErrMalformedListedRun = errors.New("the forge listed a check run without a positive id")

type checkRunID int64

type checkRunListing struct {
	TotalCount *int              `json:"total_count"`
	CheckRuns  *[]listedCheckRun `json:"check_runs"`
}

func (listed checkRunListing) soundRuns() ([]listedCheckRun, error) {
	if listed.TotalCount == nil || listed.CheckRuns == nil {
		return nil, ErrMalformedListing
	}
	runs := *listed.CheckRuns
	if *listed.TotalCount != len(runs) {
		return nil, fmt.Errorf("%w: a count of %d beside %d runs",
			ErrInconsistentListing, *listed.TotalCount, len(runs))
	}
	hasNoID := func(run listedCheckRun) bool { return run.ID <= 0 }
	if malformed := slices.IndexFunc(runs, hasNoID); malformed >= 0 {
		return nil, fmt.Errorf("%w: id %d", ErrMalformedListedRun, runs[malformed].ID)
	}
	return runs, nil
}

type listedCheckRun struct {
	ID  checkRunID `json:"id"`
	App *listedApp `json:"app"`
}

type listedApp struct {
	ID *int64 `json:"id"`
}

func (run listedCheckRun) isOwnedBy(owner appID) bool {
	return run.App != nil && run.App.ID != nil && appID(*run.App.ID) == owner
}

func ownedBy(owner appID, listed []listedCheckRun) []listedCheckRun {
	var owned []listedCheckRun
	for _, run := range listed {
		if run.isOwnedBy(owner) {
			owned = append(owned, run)
		}
	}
	return owned
}

// GitHub documents no order for listed check runs and no meaning for their ids beyond
// identity; this assumes ids are issued in increasing order, so the highest is the newest.
// started_at is not used because the schema allows it to be null:
// https://docs.github.com/en/rest/checks/runs
func newestOf(listed []listedCheckRun) listedCheckRun {
	return slices.MaxFunc(listed, func(one, other listedCheckRun) int {
		return cmp.Compare(one.ID, other.ID)
	})
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
