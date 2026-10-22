// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type RunReport struct {
	Repository GovernedRepository
	Head       CommitSHA
}
