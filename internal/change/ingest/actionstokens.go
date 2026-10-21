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

var ErrMalformedActionsTokens = errors.New("token verification settings name no https issuer or audience")

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
	Issuer   string
	Audience string
	Client   *http.Client
	Now      func() time.Time
}

type ActionsTokens struct {
	issuer   string
	audience string
	now      func() time.Time
	keys     *issuerKeys
}

func ParseActionsTokens(settings ActionsTokenSettings) (*ActionsTokens, error) {
	if err := incompletenessOf(settings); err != nil {
		return nil, err
	}
	for _, stated := range []string{settings.Issuer, settings.Audience} {
		parsed, err := url.Parse(stated)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return nil, ErrMalformedActionsTokens
		}
	}
	return &ActionsTokens{
		issuer:   settings.Issuer,
		audience: settings.Audience,
		now:      settings.Now,
		keys:     &issuerKeys{issuer: settings.Issuer, client: settings.Client, now: settings.Now},
	}, nil
}

type settingsPart struct {
	isAbsent    bool
	description string
}

func incompletenessOf(settings ActionsTokenSettings) error {
	parts := []settingsPart{
		{isAbsent: settings.Issuer == "", description: "no issuer to expect"},
		{isAbsent: settings.Audience == "", description: "no audience to expect"},
		{isAbsent: settings.Client == nil, description: "no client to reach the issuer with"},
		{isAbsent: settings.Now == nil, description: "no clock to judge a token's lifetime by"},
	}
	absent := slices.IndexFunc(parts, func(part settingsPart) bool { return part.isAbsent })
	if absent < 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrIncompleteActionsTokens, parts[absent].description)
}

func (tokens *ActionsTokens) RepositoryOf(ctx context.Context, bearer string) (domain.Repository, error) {
	keyID, err := keyIDOf(bearer)
	if err != nil {
		return domain.Repository{}, err
	}
	keys, err := tokens.keys.holding(ctx, keyID)
	if err != nil {
		return domain.Repository{}, err
	}
	verified, err := jwt.ParseString(bearer,
		jwt.WithKeySet(keys),
		jwt.WithIssuer(tokens.issuer),
		jwt.WithAudience(tokens.audience),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithRequiredClaim(jwt.NotBeforeKey),
		jwt.WithClock(jwt.ClockFunc(tokens.now)))
	if err != nil {
		return domain.Repository{}, unverifiedReasonOf(err)
	}
	return repositoryClaimedBy(verified)
}

func keyIDOf(bearer string) (string, error) {
	if len(bearer) > MaxActionsTokenBytes {
		return "", errOversizedToken
	}
	message, err := jws.ParseString(bearer)
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

func repositoryClaimedBy(verified jwt.Token) (domain.Repository, error) {
	var claimed string
	if err := verified.Get(repositoryClaim, &claimed); err != nil {
		return domain.Repository{}, errNoRepository
	}
	repository, err := domain.ParseRepository(claimed)
	if err != nil {
		return domain.Repository{}, errNotARepository
	}
	return repository, nil
}
