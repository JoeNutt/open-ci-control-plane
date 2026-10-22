// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"encoding/json"
	"fmt"
)

const redactionMarker = "[redacted]"

func writeRedactionMarker(state fmt.State) {
	_, _ = fmt.Fprint(state, redactionMarker)
}

func redactionMarkerJSON() ([]byte, error) {
	return json.Marshal(redactionMarker)
}
