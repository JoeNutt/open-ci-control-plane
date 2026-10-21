// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const forgeInstallationID = 77

const recordedInstallationToken = "ghs_16C7e42F292c6912E7710c838347Ae178B4a"

var leastPrivilegePermissions = map[string]string{
	"checks":        "write",
	"pull_requests": "read",
	"contents":      "read",
	"actions":       "write",
}

type accessTokenRequest struct {
	Permissions map[string]string `json:"permissions"`
}

func recordedGrant(expiresAt time.Time) string {
	return `{
		"token": "` + recordedInstallationToken + `",
		"expires_at": "` + expiresAt.Format(time.RFC3339) + `",
		"permissions": {"checks": "write", "pull_requests": "read", "contents": "read", "actions": "write"},
		"repository_selection": "all"
	}`
}

func tokenEndpoint(t *testing.T, now time.Time) *httptest.Server {
	t.Helper()

	routes := http.NewServeMux()
	routes.HandleFunc("POST /app/installations/{installation}/access_tokens",
		func(writer http.ResponseWriter, request *http.Request) {
			if request.PathValue("installation") != strconv.Itoa(forgeInstallationID) {
				writer.WriteHeader(http.StatusNotFound)
				return
			}
			if !isPresentedAppJWT(t, request, now) || !isLeastPrivilegeAsked(t, request) {
				writer.WriteHeader(http.StatusUnprocessableEntity)
				return
			}
			writer.WriteHeader(http.StatusCreated)
			write(t, writer, recordedGrant(now.Add(time.Hour)))
		})
	return forge(t, routes.ServeHTTP)
}

func isPresentedAppJWT(t *testing.T, request *http.Request, now time.Time) bool {
	t.Helper()

	presented, isBearer := strings.CutPrefix(request.Header.Get("Authorization"), "Bearer ")
	if !isBearer {
		t.Errorf("the token request carries no bearer App JWT")
		return false
	}
	claims := verifiedClaimsOf(t, presented, &forgeRSAKey().PublicKey)
	isIssuedByTheApp := claims.Issuer == strconv.Itoa(forgeAppID) &&
		claims.IssuedAt == now.Add(-time.Minute).Unix()
	if !isIssuedByTheApp {
		t.Errorf("the App JWT claims are %+v", claims)
	}
	return isIssuedByTheApp
}

func isLeastPrivilegeAsked(t *testing.T, request *http.Request) bool {
	t.Helper()

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var asked accessTokenRequest
	if err := decoder.Decode(&asked); err != nil {
		t.Errorf("decode the token request: %v", err)
		return false
	}
	isLeastPrivilege := maps.Equal(asked.Permissions, leastPrivilegePermissions)
	if !isLeastPrivilege {
		t.Errorf("the token request asks for %v, want exactly %v",
			asked.Permissions, leastPrivilegePermissions)
	}
	return isLeastPrivilege
}

func installationTokensAt(t *testing.T, server *httptest.Server, now time.Time) *github.InstallationTokens {
	t.Helper()

	tokens, err := github.ParseInstallationTokens(settingsAt(server), github.Installation{
		Key:   forgeAppKey(t),
		ID:    forgeInstallationID,
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("parse installation tokens: %v", err)
	}
	return tokens
}

func TestAnUnknownInstallationIsReportedAsSuchNotAsAMissingPullRequest(t *testing.T) {
	tokens := installationTokensAt(t, answering(t, http.StatusNotFound, `{"message": "Not Found"}`), mintedAt)

	_, err := github.AuthorizationOf(context.Background(), tokens)

	if !errors.Is(err, github.ErrNoSuchInstallation) || errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v and not %v",
			err, github.ErrNoSuchInstallation, domain.ErrNoSuchPullRequest)
	}
}

func TestAnInstallationTokenIsExchangedForTheAppJWTAskingExactlyLeastPrivilege(t *testing.T) {
	tokens := installationTokensAt(t, tokenEndpoint(t, mintedAt), mintedAt)

	authorization, err := github.AuthorizationOf(context.Background(), tokens)

	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if authorization != "Bearer "+recordedInstallationToken {
		t.Errorf("the presented authorization is not the exchanged installation token")
	}
}
