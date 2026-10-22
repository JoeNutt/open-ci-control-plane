// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "slices"

const pullRequestEvent = "pull_request"

var evaluatedActionsByEvent = map[string][]string{
	pullRequestEvent: {"opened", "synchronize", "reopened"},
}

func isEvaluated(event, action string) bool {
	actions, isRouted := evaluatedActionsByEvent[event]
	return isRouted && slices.Contains(actions, action)
}
