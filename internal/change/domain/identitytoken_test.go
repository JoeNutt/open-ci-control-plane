// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const presentedToken = "eyJhbGciOiJSUzI1NiJ9.eyJyZXBvc2l0b3J5Ijoib3duZXIvcmVwbyJ9.c2lnbmF0dXJl"

func TestAnIdentityTokenIsNeverFormattedWithItsValue(t *testing.T) {
	token, err := domain.ParseIdentityToken(presentedToken)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	holder := struct{ Token domain.IdentityToken }{token}

	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%p", "%T"} {
		for _, formatted := range []string{fmt.Sprintf(verb, token), fmt.Sprintf(verb, holder)} {
			if strings.Contains(formatted, presentedToken) {
				t.Errorf("%s printed the token: %s", verb, formatted)
			}
		}
	}
	encoded, err := json.Marshal(holder)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(string(encoded), presentedToken) {
		t.Errorf("JSON carried the token: %s", encoded)
	}
	if token.Reveal() != presentedToken {
		t.Errorf("Reveal returned another value")
	}
}
