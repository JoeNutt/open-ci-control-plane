// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"io"
)

type IdentityToken struct {
	value *string
}

func ParseIdentityToken(presented string) (IdentityToken, error) {
	return IdentityToken{value: &presented}, nil
}

func (IdentityToken) Format(state fmt.State, _ rune) { _, _ = io.WriteString(state, "[redacted]") }

func (IdentityToken) MarshalJSON() ([]byte, error) { return []byte(`"[redacted]"`), nil }

func (token IdentityToken) Reveal() string { return *token.value }
