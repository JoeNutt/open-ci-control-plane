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

var ErrIncompleteCheckRun = errors.New("a check run lacks a name, a title or a headline")

var ErrNotUTF8 = errors.New("a check run's text is not valid UTF-8")

var ErrCheckRunTargetNotFound = errors.New("the forge found nothing at a check-run endpoint")

type publication struct {
	locator domain.PullRequestLocator
	head    domain.CommitSHA
	name    string
	verdict checkRunVerdict
}

func parsePublication(
	locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) (publication, error) {
	if !isComplete(run) {
		return publication{}, ErrIncompleteCheckRun
	}
	if !isAllUTF8(run.Name, run.Title, run.Headline) {
		return publication{}, ErrNotUTF8
	}
	verdict, err := parseVerdict(run)
	if err != nil {
		return publication{}, err
	}
	if head == (domain.CommitSHA{}) {
		return publication{}, ErrMalformedHeadSHA
	}
	return publication{locator: locator, head: head, name: run.Name, verdict: verdict}, nil
}

func isComplete(run domain.CheckRun) bool {
	return run.Name != "" && run.Title != "" && run.Headline != ""
}

func (pending publication) String() string {
	return fmt.Sprintf("check run %q on %s at %s", pending.name, pending.locator, pending.head)
}

func isAllUTF8(texts ...string) bool {
	isMalformed := func(text string) bool { return !utf8.ValidString(text) }
	return !slices.ContainsFunc(texts, isMalformed)
}

func (pending publication) exchangeGiven(listed checkRunListing, owner appID) (exchange, error) {
	sound, err := listed.soundRuns()
	if err != nil {
		return exchange{}, err
	}
	existing := ownedBy(owner, sound)
	if len(existing) == 0 {
		return pending.creation(), nil
	}
	return pending.updateOf(newestOf(existing).ID), nil
}

func (pending publication) listing(owner appID) exchange {
	return exchange{
		method:    http.MethodGet,
		endpoint:  pending.listingEndpoint(owner),
		succeeded: http.StatusOK,
		notFound:  notFound(fmt.Sprintf("repository %s or commit %s", pending.locator.Repository(), pending.head)),
	}
}

func notFound(missing string) error {
	return fmt.Errorf("%w: %s", ErrCheckRunTargetNotFound, missing)
}

func (pending publication) creation() exchange {
	return exchange{
		method:   http.MethodPost,
		endpoint: pending.checkRunsEndpoint(),
		sent: newCheckRun{
			Name:            pending.name,
			HeadSHA:         pending.head.String(),
			checkRunVerdict: pending.verdict,
		},
		succeeded: http.StatusCreated,
		notFound:  notFound("repository " + pending.locator.Repository()),
	}
}

func (pending publication) updateOf(id checkRunID) exchange {
	return exchange{
		method:    http.MethodPatch,
		endpoint:  fmt.Sprintf("%s/%d", pending.checkRunsEndpoint(), id),
		sent:      pending.verdict,
		succeeded: http.StatusOK,
		notFound:  notFound(fmt.Sprintf("check run %d", id)),
	}
}

func (pending publication) checkRunsEndpoint() string {
	return fmt.Sprintf("/repos/%s/%s/check-runs",
		pending.locator.Owner(), pending.locator.Name())
}

func (pending publication) listingEndpoint(owner appID) string {
	query := url.Values{
		"check_name": {pending.name},
		"app_id":     {owner.String()},
		"filter":     {filterLatest},
		"per_page":   {strconv.Itoa(CheckRunsPerPage)},
	}
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?%s",
		pending.locator.Owner(), pending.locator.Name(), pending.head, query.Encode())
}
