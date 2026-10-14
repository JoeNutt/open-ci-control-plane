// SPDX-License-Identifier: GPL-3.0-or-later

package github

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
