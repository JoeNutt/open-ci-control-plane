// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"maps"
	"reflect"
)

type installationGates map[int64]ChangeGate

func parseInstallationGates(stated map[int64]ChangeGate) (installationGates, error) {
	if len(stated) == 0 {
		return nil, fmt.Errorf("%w: a gate", ErrIncompleteRoutes)
	}
	for installation, gate := range stated {
		if gate == nil || isNilPointer(gate) {
			return nil, fmt.Errorf("%w: a gate for installation %d", ErrIncompleteRoutes, installation)
		}
	}
	return installationGates(maps.Clone(stated)), nil
}

func isNilPointer(gate ChangeGate) bool {
	held := reflect.ValueOf(gate)
	return held.Kind() == reflect.Pointer && held.IsNil()
}

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
