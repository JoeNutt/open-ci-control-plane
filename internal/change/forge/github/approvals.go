// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const ReviewsPerPage = 100

const maxReviewPages = 100

const approvedReviewState = "APPROVED"

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
	return approvalsIn(reviewed, author), nil
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
	standing := standingReviewsIn(reviewed)
	approvals := make(domain.Declared[domain.Approval], 0, len(standing))
	for _, held := range standing {
		if held.state != approvedReviewState {
			continue
		}
		approvals = append(approvals, domain.Approval{
			Login:    held.reviewer,
			IsAuthor: held.reviewer == author,
		})
	}
	return approvals
}

type standingReview struct {
	reviewer string
	state    string
}

func standingReviewsIn(reviewed []review) []standingReview {
	positionOf := make(map[string]int, len(reviewed))
	var standing []standingReview
	for _, reviewing := range reviewed {
		if !replacesStandingReview[reviewing.State] {
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
	return standing
}

// https://docs.github.com/en/rest/pulls/reviews
var replacesStandingReview = map[string]bool{
	"APPROVED":          true,
	"CHANGES_REQUESTED": true,
	"DISMISSED":         true,
	"COMMENTED":         false,
	"PENDING":           false,
}
