// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

type installationGates map[int64]ChangeGate

type admission struct {
	stated payload
	gate   ChangeGate
}

func (gates installationGates) admissionOf(stated payload) (admission, bool) {
	installationID, isStated := stated.installationID()
	if !isStated {
		return admission{}, false
	}
	gate, isAllowed := gates[installationID]
	return admission{stated: stated, gate: gate}, isAllowed
}
