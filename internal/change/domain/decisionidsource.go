// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type DecisionIDSource interface {
	NextDecisionID() (DecisionID, error)
}
