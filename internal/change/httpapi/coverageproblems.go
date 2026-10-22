// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

const (
	malformedPathCode    = "malformed_path"
	malformedPathMessage = "the path segment holds a character outside its set, exceeds its length or is not one canonical segment"
)

func malformedPathProblem(segment string) refusal {
	return fieldProblem{Code: malformedPathCode, Message: malformedPathMessage, Field: segment}
}
