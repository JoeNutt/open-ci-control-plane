// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

type installationGates map[int64]ChangeGate

func (gates installationGates) gateFor(stated payload) (ChangeGate, bool) {
	installationID, isStated := stated.installationID()
	if !isStated {
		return nil, false
	}
	gate, isAllowed := gates[installationID]
	return gate, isAllowed
}
