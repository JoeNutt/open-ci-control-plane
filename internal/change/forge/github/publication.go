// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrIncompleteCheckRun = errors.New("a check run lacks a name, a title or a summary")

var ErrAmbiguousCheckRun = errors.New("the commit carries more than one check run of the name")

type publication struct {
	locator domain.PullRequestLocator
	head    commitSHA
	run     domain.CheckRun
}

func parsePublication(
	locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) (publication, error) {
	if run.Name == "" || run.Title == "" || run.Summary == "" {
		return publication{}, ErrIncompleteCheckRun
	}
	head, err := parseCommitSHA(headSHA)
	if err != nil {
		return publication{}, err
	}
	return publication{locator: locator, head: head, run: run}, nil
}

func (pending publication) exchangeGiven(listed checkRunListing) (exchange, error) {
	if listed.TotalCount > 1 {
		return exchange{}, fmt.Errorf("%w: %d named %q on %s",
			ErrAmbiguousCheckRun, listed.TotalCount, pending.run.Name, pending.head)
	}
	existing, err := listed.consistentRuns()
	if err != nil {
		return exchange{}, err
	}
	if len(existing) == 0 {
		return pending.creation(), nil
	}
	return pending.updateOf(existing[0].ID), nil
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
