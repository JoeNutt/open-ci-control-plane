// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const ReviewsPerPage = 100

// A pull request's review history has no documented ceiling, so this bound is this
// project's: 100 pages of ReviewsPerPage is more reviews than any change under
// NFR-5's bounds can have attracted.
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
	standing := make(map[string]string, len(reviewed))
	var reviewers []string
	for _, reviewing := range reviewed {
		if !decidesApproval[reviewing.State] {
			continue
		}
		if _, reviewed := standing[reviewing.User.Login]; !reviewed {
			reviewers = append(reviewers, reviewing.User.Login)
		}
		standing[reviewing.User.Login] = reviewing.State
	}
	approvals := make(domain.Declared[domain.Approval], 0, len(reviewers))
	for _, reviewer := range reviewers {
		if standing[reviewer] != approvedReviewState {
			continue
		}
		approvals = append(approvals, domain.Approval{
			Login:    reviewer,
			IsAuthor: reviewer == author,
		})
	}
	return approvals
}

// A review of COMMENTED or PENDING leaves a reviewer's earlier approval or request
// for changes standing; the other states replace it:
// https://docs.github.com/en/rest/pulls/reviews
var decidesApproval = map[string]bool{
	"APPROVED":          true,
	"CHANGES_REQUESTED": true,
	"DISMISSED":         true,
	"COMMENTED":         false,
	"PENDING":           false,
}
