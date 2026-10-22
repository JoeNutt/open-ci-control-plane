// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type PolicyCommit struct {
	Repository PolicyRepository
	Commit     CommitSHA
}
