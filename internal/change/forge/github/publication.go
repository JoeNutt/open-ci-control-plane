// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrIncompleteCheckRun = errors.New("a check run lacks a name, a title or a summary")

// GitHub accepts at most 65535 characters in each of output.summary and output.text:
// https://docs.github.com/en/rest/checks/runs
const MaxOutputCharacters = 65535

var ErrOutputTooLong = errors.New("a check run's summary or text is longer than the forge accepts")

var ErrAmbiguousCheckRun = errors.New("the commit carries more than one check run of the name")

type publication struct {
	locator domain.PullRequestLocator
	head    commitSHA
	name    string
	verdict checkRunVerdict
}

func parsePublication(
	locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) (publication, error) {
	if run.Name == "" || run.Title == "" || run.Summary == "" {
		return publication{}, ErrIncompleteCheckRun
	}
	if !fitsOutput(run.Summary) || !fitsOutput(run.Text) {
		return publication{}, ErrOutputTooLong
	}
	verdict, err := parseVerdict(run)
	if err != nil {
		return publication{}, err
	}
	head, err := parseCommitSHA(headSHA)
	if err != nil {
		return publication{}, err
	}
	return publication{locator: locator, head: head, name: run.Name, verdict: verdict}, nil
}

func fitsOutput(field string) bool {
	return utf8.RuneCountInString(field) <= MaxOutputCharacters
}

func (pending publication) exchangeGiven(listed checkRunListing) (exchange, error) {
	if listed.TotalCount > 1 {
		return exchange{}, fmt.Errorf("%w: %d named %q on %s",
			ErrAmbiguousCheckRun, listed.TotalCount, pending.name, pending.head)
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
			Name:            pending.name,
			HeadSHA:         string(pending.head),
			checkRunVerdict: pending.verdict,
		},
		succeeded: http.StatusCreated,
	}
}

func (pending publication) updateOf(id checkRunID) exchange {
	return exchange{
		method:    http.MethodPatch,
		endpoint:  fmt.Sprintf("%s/%d", pending.checkRunsEndpoint(), id),
		sent:      pending.verdict,
		succeeded: http.StatusOK,
	}
}

func (pending publication) checkRunsEndpoint() string {
	return fmt.Sprintf("/repos/%s/%s/check-runs",
		pending.locator.Owner(), pending.locator.Name())
}

func (pending publication) listingEndpoint() string {
	query := url.Values{"check_name": {pending.name}}
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?%s",
		pending.locator.Owner(), pending.locator.Name(), pending.head, query.Encode())
}
