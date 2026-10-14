// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"io"
)

type Credential string

func (credential Credential) String() string { return "[redacted]" }

func (credential Credential) GoString() string { return `"[redacted]"` }

func (credential Credential) Format(state fmt.State, verb rune) {
	if verb == 'v' && state.Flag('#') {
		_, _ = io.WriteString(state, credential.GoString())
		return
	}
	_, _ = io.WriteString(state, credential.String())
}
