// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const ReviewsPerPage = 100

// A pull request's review history has no documented ceiling, so this bound is this
// project's: 100 pages of ReviewsPerPage is more reviews than any change under
// NFR-5's bounds can have attracted.
const maxReviewPages = 100

var ErrTooManyReviewPages = errors.New("the forge offered more pages of reviews than the bound allows")

const approvedReviewState = "APPROVED"

func (source ChangeSource) approvalsOf(
	ctx context.Context, locator domain.PullRequestLocator, author string,
) (domain.Declared[domain.Approval], error) {
	var reviewed []review
	for page := 1; page <= maxReviewPages; page++ {
		onPage, err := jsonAt[[]review](ctx, source, reviewsEndpoint(locator, page))
		if err != nil {
			return nil, err
		}
		reviewed = append(reviewed, onPage...)
		if len(onPage) < ReviewsPerPage {
			return approvalsIn(reviewed, author), nil
		}
	}
	return nil, fmt.Errorf("%w: %d pages of %d",
		ErrTooManyReviewPages, maxReviewPages, ReviewsPerPage)
}

func reviewsEndpoint(locator domain.PullRequestLocator, page int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews?per_page=%d&page=%d",
		locator.Owner(), locator.Name(), locator.Number(), ReviewsPerPage, page)
}

type review struct {
	User  account `json:"user"`
	State string  `json:"state"`
}

func approvalsIn(reviewed []review, author string) domain.Declared[domain.Approval] {
	approvals := make(domain.Declared[domain.Approval], 0, len(reviewed))
	for _, reviewing := range reviewed {
		if reviewing.State != approvedReviewState {
			continue
		}
		approvals = append(approvals, domain.Approval{
			Login:    reviewing.User.Login,
			IsAuthor: reviewing.User.Login == author,
		})
	}
	return approvals
}
