// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"
)

const redactionMarker = "[redacted]"

type Credential struct {
	token *string
}

func NewCredential(token string) Credential { return Credential{token: &token} }

func (credential Credential) isPresent() bool {
	return credential.token != nil && *credential.token != ""
}

func (credential Credential) bearer() string { return "Bearer " + *credential.token }

func (credential Credential) authorization(context.Context) (Credential, error) {
	return credential, nil
}

func (Credential) Format(state fmt.State, _ rune) { writeRedactionMarker(state) }

func (Credential) MarshalJSON() ([]byte, error) { return []byte(`"[redacted]"`), nil }

func writeRedactionMarker(state fmt.State) {
	_, _ = fmt.Fprint(state, redactionMarker)
}
