// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrNoSuchCommit = errors.New("the forge reports no such commit in the repository")

type PolicyCommit struct {
	Repository PolicyRepository
	Commit     CommitSHA
}
