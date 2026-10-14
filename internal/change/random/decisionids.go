// SPDX-License-Identifier: GPL-3.0-or-later

package random

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

type DecisionIDs struct{}

func (DecisionIDs) NextDecisionID() (domain.DecisionID, error) {
	return domain.ParseDecisionID("dec_00000000000000000000000000000000")
}
