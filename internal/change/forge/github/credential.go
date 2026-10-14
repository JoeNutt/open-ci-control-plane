// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"io"
)

type Credential struct {
	token *string
}

func NewCredential(token string) Credential { return Credential{token: &token} }

func (credential Credential) isPresent() bool {
	return credential.token != nil && *credential.token != ""
}

func (credential Credential) bearer() string { return "Bearer " + *credential.token }

func (credential Credential) String() string { return "[redacted]" }

func (credential Credential) GoString() string { return `"[redacted]"` }

func (credential Credential) Format(state fmt.State, verb rune) {
	formatRedacted(state, verb, credential)
}

type redactedDescription interface {
	fmt.Stringer
	fmt.GoStringer
}

// Format has no error return, and the fmt.State that fmt passes it writes to fmt's own
// buffer, whose Write always returns a nil error: https://pkg.go.dev/fmt#Formatter
func formatRedacted(state fmt.State, verb rune, holder redactedDescription) {
	if verb == 'v' && state.Flag('#') {
		_, _ = io.WriteString(state, holder.GoString())
		return
	}
	_, _ = io.WriteString(state, holder.String())
}
