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

var ErrIncompleteGrant = errors.New("incomplete")

type Installation struct {
	Key   AppKey
	ID    int64
	Clock func() time.Time
}

type InstallationTokens struct {
	asApp        connection
	installation installationID
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
	return &InstallationTokens{asApp: asApp, installation: installed}, nil
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
	if granted.Token == "" || granted.ExpiresAt.IsZero() {
		return Credential{}, ErrIncompleteGrant
	}
	return NewCredential(granted.Token), nil
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
