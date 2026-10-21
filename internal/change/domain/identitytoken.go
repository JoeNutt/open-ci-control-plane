// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const redacted = "[redacted]"

var ErrAbsentIdentityToken = errors.New("no identity token was presented")

type IdentityToken struct {
	value *string
}

func ParseIdentityToken(presented string) (IdentityToken, error) {
	if presented == "" {
		return IdentityToken{}, ErrAbsentIdentityToken
	}
	return IdentityToken{value: &presented}, nil
}

func (IdentityToken) Format(state fmt.State, _ rune) { _, _ = io.WriteString(state, redacted) }

func (IdentityToken) MarshalJSON() ([]byte, error) { return json.Marshal(redacted) }

func (token IdentityToken) Reveal() string { return *token.value }
