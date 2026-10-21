// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type Verdict string

const (
	VerdictAdmitted       Verdict = "admitted"
	VerdictRefused        Verdict = "refused"
	VerdictReviewRequired Verdict = "review_required"
)
