// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

const (
	contentTypeHeader = "Content-Type"
	jsonMediaType     = "application/json"
	unwrittenAnswer   = "the answer to a delivery could not be written"
)

const (
	missingHeaderCode      = "missing_header"
	missingHeaderMessage   = "the delivery carries no such header"
	malformedHeaderCode    = "malformed_header"
	malformedHeaderMessage = "the header exceeds its length or holds a character outside its set"
)

type fieldProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

func missingHeaderProblem(header string) fieldProblem {
	return fieldProblem{Code: missingHeaderCode, Message: missingHeaderMessage, Field: header}
}

func malformedHeaderProblem(header string) fieldProblem {
	return fieldProblem{Code: malformedHeaderCode, Message: malformedHeaderMessage, Field: header}
}
