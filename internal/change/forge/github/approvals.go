// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const ReviewsPerPage = 100

const maxReviewPages = 100

type reviewState string

const (
	reviewApproved         reviewState = "APPROVED"
	reviewChangesRequested reviewState = "CHANGES_REQUESTED"
	reviewDismissed        reviewState = "DISMISSED"
	reviewCommented        reviewState = "COMMENTED"
	reviewPending          reviewState = "PENDING"
)

var ErrUnknownReviewState = errors.New("the forge reports a review state the adapter cannot classify")

func (source ChangeSource) approvalsOf(
	ctx context.Context, locator domain.PullRequestLocator, author string,
) (domain.Declared[domain.Approval], error) {
	reviewed, err := acrossPages[review](ctx, source, paging{
		endpointAt: func(page int) string { return reviewsEndpoint(locator, page) },
		perPage:    ReviewsPerPage,
		maxPages:   maxReviewPages,
	})
	if err != nil {
		return nil, err
	}
	return approvalsIn(reviewed, author)
}

func reviewsEndpoint(locator domain.PullRequestLocator, page int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews?per_page=%d&page=%d",
		locator.Owner(), locator.Name(), locator.Number(), ReviewsPerPage, page)
}

type review struct {
	User  account     `json:"user"`
	State reviewState `json:"state"`
}

func approvalsIn(
	reviewed []review, author string,
) (domain.Declared[domain.Approval], error) {
	standing, err := standingReviewsIn(reviewed)
	if err != nil {
		return nil, err
	}
	approvals := make(domain.Declared[domain.Approval], 0, len(standing))
	for _, held := range standing {
		if held.state != reviewApproved {
			continue
		}
		approvals = append(approvals, domain.Approval{
			Login:    held.reviewer,
			IsAuthor: held.reviewer == author,
		})
	}
	return approvals, nil
}

type standingReview struct {
	reviewer string
	state    reviewState
}

func standingReviewsIn(reviewed []review) ([]standingReview, error) {
	positionOf := make(map[string]int, len(reviewed))
	var standing []standingReview
	for _, reviewing := range reviewed {
		replaces, declared := replacesStandingReview[reviewing.State]
		if !declared {
			return nil, fmt.Errorf("%w: %q", ErrUnknownReviewState, reviewing.State)
		}
		if !replaces {
			continue
		}
		position, reviewedBefore := positionOf[reviewing.User.Login]
		if !reviewedBefore {
			positionOf[reviewing.User.Login] = len(standing)
			standing = append(standing, standingReview{
				reviewer: reviewing.User.Login,
				state:    reviewing.State,
			})
			continue
		}
		standing[position].state = reviewing.State
	}
	return standing, nil
}

var replacesStandingReview = map[reviewState]bool{
	reviewApproved:         true,
	reviewChangesRequested: true,
	reviewDismissed:        true,
	reviewCommented:        false,
	reviewPending:          false,
}
