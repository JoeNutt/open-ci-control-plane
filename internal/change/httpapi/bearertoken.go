// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	bearerScheme        = "Bearer"
	schemeSeparator     = " "
	b64tokenCharacters  = lowerASCIILetters + upperASCIILetters + asciiDigits + "-._~+/="
	authorizationHeader = "Authorization"
)

var errNotABearerToken = fmt.Errorf("%w: the authorization is not one bearer token", domain.ErrUnverifiedIdentityToken)

func bearerTokenIn(header string) (domain.IdentityToken, error) {
	scheme, token, isSeparated := strings.Cut(header, schemeSeparator)
	if !isSeparated || !strings.EqualFold(scheme, bearerScheme) || !isB64token(token) {
		return domain.IdentityToken{}, errNotABearerToken
	}
	return domain.ParseIdentityToken(token)
}

func isB64token(stated string) bool {
	return stated != "" && strings.Trim(stated, b64tokenCharacters) == ""
}
