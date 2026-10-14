// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.CheckRunPublisher = CheckRuns{}

var ErrMalformedHeadSHA = errors.New("the head commit is not a 40-digit lowercase hex SHA")

type CheckRuns struct {
	forge connection
}

func ParseCheckRuns(settings Settings) (CheckRuns, error) {
	forge, err := parseConnection(settings)
	if err != nil {
		return CheckRuns{}, err
	}
	return CheckRuns{forge: forge}, nil
}

func (runs CheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) error {
	if len(headSHA) != 40 || strings.Trim(headSHA, "0123456789abcdef") != "" {
		return fmt.Errorf("%w: %q", ErrMalformedHeadSHA, headSHA)
	}
	_, err := jsonAt[checkRunListing](ctx, runs.forge, checkRunsOnCommitEndpoint(locator, headSHA, run.Name))
	if err != nil {
		return err
	}
	_, err = exchanged[struct{}](ctx, runs.forge, exchange{
		method:    http.MethodPost,
		endpoint:  checkRunsEndpoint(locator),
		sent:      newCheckRun{Name: run.Name, HeadSHA: headSHA, checkRunVerdict: verdictOf(run)},
		succeeded: http.StatusCreated,
	})
	return err
}
