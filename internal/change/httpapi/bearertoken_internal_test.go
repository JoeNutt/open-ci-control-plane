// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const presentedToken = "eyJhbGciOiJSUzI1NiJ9.claims.signature"

func TestABearerAuthorizationPresentsItsToken(t *testing.T) {
	for _, header := range []string{"Bearer " + presentedToken, "bearer " + presentedToken, "BEARER " + presentedToken} {
		token, err := bearerTokenIn(header)

		if err != nil || token.Reveal() != presentedToken {
			t.Errorf("%q: presented %v, want the token", header, err)
		}
	}
}

func TestAnAuthorizationThatIsNotExactlyOneBearerTokenPresentsNone(t *testing.T) {
	for form, header := range map[string]string{
		"absent":               "",
		"another scheme":       "Basic " + presentedToken,
		"no scheme":            presentedToken,
		"an empty token":       "Bearer ",
		"a scheme alone":       "Bearer",
		"two tokens":           "Bearer " + presentedToken + " " + presentedToken,
		"a doubled separator":  "Bearer  " + presentedToken,
		"a tab for the space":  "Bearer\t" + presentedToken,
		"a scheme with a tail": "Bearerx " + presentedToken,
	} {
		t.Run(form, func(t *testing.T) {
			_, err := bearerTokenIn(header)

			if !errors.Is(err, domain.ErrUnverifiedIdentityToken) {
				t.Errorf("error is %v, want %v", err, domain.ErrUnverifiedIdentityToken)
			}
		})
	}
}
