// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Installation struct {
	Key   AppKey
	ID    int64
	Clock func() time.Time
}

type InstallationTokens struct {
	settings     Settings
	installation Installation
}

func ParseInstallationTokens(settings Settings, installation Installation) (*InstallationTokens, error) {
	return &InstallationTokens{settings: settings, installation: installation}, nil
}

type perms struct {
	Checks       string `json:"checks"`
	PullRequests string `json:"pull_requests"`
	Contents     string `json:"contents"`
	Actions      string `json:"actions"`
}

type tokenReq struct {
	Permissions perms `json:"permissions"`
}

type grant struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (tokens *InstallationTokens) authorization(ctx context.Context) (Credential, error) {
	jwt, err := appJWTAt(tokens.installation.Key, appID(tokens.settings.AppID), tokens.installation.Clock())
	if err != nil {
		return Credential{}, err
	}
	forge := connection{settings: tokens.settings, token: NewCredential(jwt)}
	g, err := exchanged[grant](ctx, forge, exchange{
		method:    http.MethodPost,
		endpoint:  fmt.Sprintf("/app/installations/%d/access_tokens", tokens.installation.ID),
		sent:      tokenReq{Permissions: perms{"write", "read", "read", "write"}},
		succeeded: http.StatusCreated,
	})
	if err != nil {
		return Credential{}, err
	}
	return NewCredential(g.Token), nil
}
