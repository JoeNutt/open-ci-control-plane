// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrIncompleteCheckRun = errors.New("a check run lacks a name, a title or a summary")

var ErrNotUTF8 = errors.New("a check run's text is not valid UTF-8")

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
	if run.Name == "" {
		return publication{}, ErrIncompleteCheckRun
	}
	if !isAllUTF8(run.Name, run.Title, run.Summary, run.Text) {
		return publication{}, ErrNotUTF8
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

func (pending publication) String() string {
	return fmt.Sprintf("check run %q on %s at %s", pending.name, pending.locator, pending.head)
}

func isAllUTF8(texts ...string) bool {
	isMalformed := func(text string) bool { return !utf8.ValidString(text) }
	return !slices.ContainsFunc(texts, isMalformed)
}

func (pending publication) exchangeGiven(listed checkRunListing) (exchange, error) {
	if listed.TotalCount > 1 {
		return exchange{}, fmt.Errorf("%w: %d of them", ErrAmbiguousCheckRun, listed.TotalCount)
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

func (pending publication) listingEndpoint(owner int64) string {
	query := url.Values{
		"check_name": {pending.name},
		"app_id":     {strconv.FormatInt(owner, 10)},
		"filter":     {filterLatest},
	}
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?%s",
		pending.locator.Owner(), pending.locator.Name(), pending.head, query.Encode())
}
