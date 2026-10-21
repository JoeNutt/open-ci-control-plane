// SPDX-License-Identifier: GPL-3.0-or-later

package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	discoveryPath     = "/.well-known/openid-configuration"
	repositoryClaim   = "repository"
	maxDiscoveryBytes = 64 << 10
)

var ErrUnverifiedToken = errors.New("token is absent, expired or not verifiable")

var (
	errUnverifiableSignature = fmt.Errorf("%w: its signature or form does not verify", ErrUnverifiedToken)
	errForeignIssuer         = fmt.Errorf("%w: another issuer signed it", ErrUnverifiedToken)
	errForeignAudience       = fmt.Errorf("%w: it was minted for another audience", ErrUnverifiedToken)
)

type unverifiedReason struct {
	cause  error
	reason error
}

var unverifiedReasons = []unverifiedReason{
	{cause: jwt.InvalidIssuerError(), reason: errForeignIssuer},
	{cause: jwt.InvalidAudienceError(), reason: errForeignAudience},
}

type ActionsTokenSettings struct {
	Issuer   string
	Audience string
	Client   *http.Client
	Now      func() time.Time
}

type ActionsTokens struct {
	settings ActionsTokenSettings
}

type discoveryDocument struct {
	KeySets string `json:"jwks_uri"`
}

func ParseActionsTokens(settings ActionsTokenSettings) (*ActionsTokens, error) {
	return &ActionsTokens{settings: settings}, nil
}

func (tokens *ActionsTokens) RepositoryOf(ctx context.Context, bearer string) (domain.Repository, error) {
	keys, err := tokens.signingKeys(ctx)
	if err != nil {
		return domain.Repository{}, err
	}
	verified, err := jwt.ParseString(bearer,
		jwt.WithKeySet(keys),
		jwt.WithIssuer(tokens.settings.Issuer),
		jwt.WithAudience(tokens.settings.Audience),
		jwt.WithClock(jwt.ClockFunc(tokens.settings.Now)))
	if err != nil {
		return domain.Repository{}, unverifiedReasonOf(err)
	}
	var claimed string
	if err := verified.Get(repositoryClaim, &claimed); err != nil {
		return domain.Repository{}, fmt.Errorf("read the %s claim: %w", repositoryClaim, err)
	}
	return domain.ParseRepository(claimed)
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

func (tokens *ActionsTokens) signingKeys(ctx context.Context) (jwk.Set, error) {
	location, err := tokens.keySetLocation(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := jwk.Fetch(ctx, location, jwk.WithHTTPClient(tokens.settings.Client))
	if err != nil {
		return nil, fmt.Errorf("fetch the issuer's key set: %w", err)
	}
	return keys, nil
}

func (tokens *ActionsTokens) keySetLocation(ctx context.Context) (string, error) {
	asked, err := http.NewRequestWithContext(ctx, http.MethodGet, tokens.settings.Issuer+discoveryPath, nil)
	if err != nil {
		return "", fmt.Errorf("ask the issuer for its configuration: %w", err)
	}
	answer, err := tokens.settings.Client.Do(asked)
	if err != nil {
		return "", fmt.Errorf("ask the issuer for its configuration: %w", err)
	}
	document, err := boundedBodyOf(answer)
	if err != nil {
		return "", fmt.Errorf("read the issuer's configuration: %w", err)
	}
	var discovered discoveryDocument
	if err := json.Unmarshal(document, &discovered); err != nil {
		return "", fmt.Errorf("decode the issuer's configuration: %w", err)
	}
	return discovered.KeySets, nil
}

var errOversizedConfiguration = fmt.Errorf("the issuer's configuration exceeds %d bytes", maxDiscoveryBytes)

func boundedBodyOf(answer *http.Response) ([]byte, error) {
	body, readFailure := io.ReadAll(io.LimitReader(answer.Body, maxDiscoveryBytes+1))
	if err := errors.Join(readFailure, answer.Body.Close()); err != nil {
		return nil, err
	}
	if answer.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the issuer answered %d", answer.StatusCode)
	}
	if len(body) > maxDiscoveryBytes {
		return nil, errOversizedConfiguration
	}
	return body, nil
}
