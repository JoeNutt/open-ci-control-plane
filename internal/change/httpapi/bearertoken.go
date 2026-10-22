// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var errNotABearerToken = fmt.Errorf("%w: the authorization is not one bearer token", domain.ErrUnverifiedIdentityToken)

func bearerTokenIn(header string) (domain.IdentityToken, error) {
	scheme, token, isSeparated := strings.Cut(header, " ")
	if !isSeparated || !strings.EqualFold(scheme, "Bearer") || token == "" ||
		strings.Trim(token, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~+/=") != "" {
		return domain.IdentityToken{}, errNotABearerToken
	}
	return domain.ParseIdentityToken(token)
}
