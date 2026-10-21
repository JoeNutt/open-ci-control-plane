// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrNoSuchInstallation = errors.New("the forge found no App installation with the configured installation id")

type Installation struct {
	Key   AppKey
	ID    int64
	Clock func() time.Time
}

type InstallationTokens struct {
	asApp        connection
	installation int64
}

var ErrMalformedInstallationID = errors.New("bad installation id")

var ErrNoAppKey = errors.New("no key")

var ErrNoClock = errors.New("no clock")

func ParseInstallationTokens(settings Settings, installation Installation) (*InstallationTokens, error) {
	if settings.AppID <= 0 {
		return nil, ErrMalformedAppID
	}
	if installation.ID <= 0 {
		return nil, ErrMalformedInstallationID
	}
	if installation.Key.key == nil {
		return nil, ErrNoAppKey
	}
	if installation.Clock == nil {
		return nil, ErrNoClock
	}
	asApp, err := parseConnection(settings, appJWTIssuer{
		key:   installation.Key,
		app:   appID(settings.AppID),
		clock: installation.Clock,
	})
	if err != nil {
		return nil, err
	}
	return &InstallationTokens{asApp: asApp, installation: installation.ID}, nil
}

func (tokens *InstallationTokens) authorization(ctx context.Context) (Credential, error) {
	granted, err := exchanged[accessTokenGrant](ctx, tokens.asApp, exchange{
		method:    http.MethodPost,
		endpoint:  tokens.accessTokensEndpoint(),
		sent:      accessTokenRequest{Permissions: leastPrivilege},
		succeeded: http.StatusCreated,
		notFound:  ErrNoSuchInstallation,
	})
	if err != nil {
		return Credential{}, err
	}
	return NewCredential(granted.Token), nil
}

func (tokens *InstallationTokens) accessTokensEndpoint() string {
	return fmt.Sprintf("/app/installations/%d/access_tokens", tokens.installation)
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
