// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var ErrNoSuchInstallation = errors.New("the forge found no App installation with the configured installation id")

var ErrIncompleteGrant = errors.New("the forge granted an installation token with no token or no expiry")

var ErrExpiredGrant = errors.New("the forge granted an installation token already past its expiry")

type Installation struct {
	Key   AppKey
	ID    int64
	Clock func() time.Time
}

type InstallationTokens struct {
	asApp        connection
	installation installationID
	mutex        sync.Mutex
	held         installationToken
	clock        func() time.Time
}

func ParseInstallationTokens(settings Settings, installation Installation) (*InstallationTokens, error) {
	issuer, err := parseAppJWTIssuer(settings, installation)
	if err != nil {
		return nil, err
	}
	installed, err := parseInstallationID(installation.ID)
	if err != nil {
		return nil, err
	}
	asApp, err := parseConnection(settings, issuer)
	if err != nil {
		return nil, err
	}
	return &InstallationTokens{asApp: asApp, installation: installed, clock: installation.Clock}, nil
}

func (tokens *InstallationTokens) isPresent() bool { return tokens != nil }

func (tokens *InstallationTokens) authorization(ctx context.Context) (Credential, error) {
	tokens.mutex.Lock()
	defer tokens.mutex.Unlock()

	now := tokens.clock()
	if tokens.held.isUsableAt(now) {
		return tokens.held.credential, nil
	}
	minted, err := tokens.exchangedAt(ctx, now)
	if err != nil {
		return Credential{}, err
	}
	tokens.held = minted
	return minted.credential, nil
}

func (tokens *InstallationTokens) exchangedAt(ctx context.Context, now time.Time) (installationToken, error) {
	granted, err := exchanged[accessTokenGrant](ctx, tokens.asApp, exchange{
		method:    http.MethodPost,
		endpoint:  tokens.accessTokensEndpoint(),
		sent:      accessTokenRequest{Permissions: leastPrivilege},
		succeeded: http.StatusCreated,
		notFound:  ErrNoSuchInstallation,
	})
	if err != nil {
		return installationToken{}, fmt.Errorf("exchange the App JWT for an installation token: %w", err)
	}
	return parseGrantAt(granted, now)
}

func (tokens *InstallationTokens) accessTokensEndpoint() string {
	return fmt.Sprintf("/app/installations/%s/access_tokens", tokens.installation)
}

const (
	permissionRead  = "read"
	permissionWrite = "write"
)

type tokenPermissions struct {
	Checks       string `json:"checks"`
	PullRequests string `json:"pull_requests"`
	Contents     string `json:"contents"`
	Actions      string `json:"actions"`
}

var leastPrivilege = tokenPermissions{
	Checks:       permissionWrite,
	PullRequests: permissionRead,
	Contents:     permissionRead,
	Actions:      permissionWrite,
}

type accessTokenRequest struct {
	Permissions tokenPermissions `json:"permissions"`
}

type accessTokenGrant struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

const tokenRefreshLead = 5 * time.Minute

type installationToken struct {
	credential Credential
	expiresAt  time.Time
}

func parseGrantAt(granted accessTokenGrant, now time.Time) (installationToken, error) {
	if granted.Token == "" || granted.ExpiresAt.IsZero() {
		return installationToken{}, ErrIncompleteGrant
	}
	if !granted.ExpiresAt.After(now) {
		return installationToken{}, fmt.Errorf("%w: expired at %s, now %s",
			ErrExpiredGrant, granted.ExpiresAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	return installationToken{credential: NewCredential(granted.Token), expiresAt: granted.ExpiresAt}, nil
}

func (token installationToken) isUsableAt(now time.Time) bool {
	return token.credential.isPresent() && now.Before(token.expiresAt.Add(-tokenRefreshLead))
}
