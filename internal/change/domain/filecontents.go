// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type FileAtCommit struct {
	Repository string
	Commit     CommitSHA
	Path       string
}
