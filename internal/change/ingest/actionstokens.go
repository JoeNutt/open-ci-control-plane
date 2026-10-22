// SPDX-License-Identifier: GPL-3.0-or-later

package ingest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	MaxActionsTokenBytes = 16 << 10
	repositoryClaim      = "repository"
)

var ErrIncompleteActionsTokens = errors.New("token verification is missing a part it cannot work without")

var ErrUnverifiedToken = errors.New("token is absent, expired or not verifiable")

var (
	errUnverifiableSignature = fmt.Errorf("%w: its signature or form does not verify", ErrUnverifiedToken)
	errForeignIssuer         = fmt.Errorf("%w: another issuer signed it", ErrUnverifiedToken)
	errForeignAudience       = fmt.Errorf("%w: it was minted for another audience", ErrUnverifiedToken)
	errExpired               = fmt.Errorf("%w: it has expired", ErrUnverifiedToken)
	errNotYetValid           = fmt.Errorf("%w: it is not yet valid", ErrUnverifiedToken)
	errNoLifetime            = fmt.Errorf("%w: it states no expiry or not-before", ErrUnverifiedToken)
	errNoRepository          = fmt.Errorf("%w: it names no repository", ErrUnverifiedToken)
	errNotARepository        = fmt.Errorf("%w: its repository claim is not an owner/name pair", ErrUnverifiedToken)
	errOversizedToken        = fmt.Errorf("%w: it exceeds %d bytes", ErrUnverifiedToken, MaxActionsTokenBytes)
)

type unverifiedReason struct {
	cause  error
	reason error
}

var unverifiedReasons = []unverifiedReason{
	{cause: jwt.InvalidIssuerError(), reason: errForeignIssuer},
	{cause: jwt.InvalidAudienceError(), reason: errForeignAudience},
	{cause: jwt.TokenExpiredError(), reason: errExpired},
	{cause: jwt.TokenNotYetValidError(), reason: errNotYetValid},
	{cause: jwt.MissingRequiredClaimError(), reason: errNoLifetime},
}

type ActionsTokenSettings struct {
	Issuer   domain.HTTPSOrigin
	Audience domain.HTTPSOrigin
	Client   *http.Client
	Now      func() time.Time
}

type ActionsTokens struct {
	issuer   *url.URL
	audience domain.HTTPSOrigin
	now      func() time.Time
	keys     *issuerKeys
}

func ParseActionsTokens(settings ActionsTokenSettings) (*ActionsTokens, error) {
	if err := incompletenessOf(settings); err != nil {
		return nil, err
	}
	issuer, err := url.Parse(settings.Issuer.String())
	if err != nil {
		return nil, err
	}
	return &ActionsTokens{
		issuer:   issuer,
		audience: settings.Audience,
		now:      settings.Now,
		keys:     &issuerKeys{issuer: issuer, client: settings.Client, now: settings.Now},
	}, nil
}

type settingsPart struct {
	isAbsent    bool
	description string
}

func incompletenessOf(settings ActionsTokenSettings) error {
	parts := []settingsPart{
		{isAbsent: settings.Issuer == domain.HTTPSOrigin{}, description: "no issuer to expect"},
		{isAbsent: settings.Audience == domain.HTTPSOrigin{}, description: "no audience to expect"},
		{isAbsent: settings.Client == nil, description: "no client to reach the issuer with"},
		{isAbsent: hasNoTimeout(settings.Client), description: "no timeout on the client"},
		{isAbsent: settings.Now == nil, description: "no clock to judge a token's lifetime by"},
	}
	absent := slices.IndexFunc(parts, func(part settingsPart) bool { return part.isAbsent })
	if absent < 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrIncompleteActionsTokens, parts[absent].description)
}

func hasNoTimeout(client *http.Client) bool {
	return client == nil || client.Timeout <= 0
}

func (tokens *ActionsTokens) RepositoryOf(ctx context.Context, token domain.IdentityToken) (domain.GovernedRepository, error) {
	keyID, err := keyIDOf(token)
	if err != nil {
		return domain.GovernedRepository{}, err
	}
	keys, err := tokens.keys.holding(ctx, keyID)
	if err != nil {
		return domain.GovernedRepository{}, err
	}
	verified, err := jwt.ParseString(token.Reveal(),
		jwt.WithKeySet(keys),
		jwt.WithIssuer(tokens.issuer.String()),
		jwt.WithAudience(tokens.audience.String()),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithRequiredClaim(jwt.NotBeforeKey),
		jwt.WithClock(jwt.ClockFunc(tokens.now)))
	if err != nil {
		return domain.GovernedRepository{}, unverifiedReasonOf(err)
	}
	return repositoryClaimedBy(verified)
}

func keyIDOf(token domain.IdentityToken) (string, error) {
	if len(token.Reveal()) > MaxActionsTokenBytes {
		return "", errOversizedToken
	}
	message, err := jws.ParseString(token.Reveal())
	if err != nil || len(message.Signatures()) != 1 {
		return "", errUnverifiableSignature
	}
	keyID, isNamed := message.Signatures()[0].ProtectedHeaders().KeyID()
	if !isNamed {
		return "", errUnverifiableSignature
	}
	return keyID, nil
}

func unverifiedReasonOf(failure error) error {
	matching := slices.IndexFunc(unverifiedReasons, func(row unverifiedReason) bool {
		return errors.Is(failure, row.cause)
	})
	if matching < 0 {
		return errUnverifiableSignature
	}
	return unverifiedReasons[matching].reason
}

func repositoryClaimedBy(verified jwt.Token) (domain.GovernedRepository, error) {
	var claimed string
	if err := verified.Get(repositoryClaim, &claimed); err != nil {
		return domain.GovernedRepository{}, errNoRepository
	}
	repository, err := domain.ParseGovernedRepository(claimed)
	if err != nil {
		return domain.GovernedRepository{}, errNotARepository
	}
	return repository, nil
}
