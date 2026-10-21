// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

const (
	pullRequestEvent = "pull_request"
	openedAction     = "opened"
)

func isEvaluated(event, action string) bool {
	return event == pullRequestEvent && (action == openedAction || action == "synchronize" || action == "reopened")
}
