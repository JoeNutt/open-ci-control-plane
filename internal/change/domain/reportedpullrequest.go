// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ReportedPullRequest struct {
	PullRequest PullRequest
	Tips        BranchTips
}
