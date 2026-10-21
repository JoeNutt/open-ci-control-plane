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
	"sync"
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

const grantedLifetime = time.Hour

func grantedToken(serial int) string {
	return recordedInstallationToken + "-" + strconv.Itoa(serial)
}

func recordedGrant(token string, expiresAt time.Time) string {
	return `{
		"token": "` + token + `",
		"expires_at": "` + expiresAt.Format(time.RFC3339) + `",
		"permissions": {"checks": "write", "pull_requests": "read", "contents": "read", "actions": "write"},
		"repository_selection": "all"
	}`
}

type forgeClock struct {
	mutex sync.Mutex
	now   time.Time
}

func clockAt(now time.Time) *forgeClock { return &forgeClock{now: now} }

func (clock *forgeClock) Now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	return clock.now
}

func (clock *forgeClock) advance(by time.Duration) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.now = clock.now.Add(by)
}

type tokenForge struct {
	t          *testing.T
	clock      *forgeClock
	server     *httptest.Server
	mutex      sync.Mutex
	exchanges  int
	isRefusing bool
	repository http.Handler
}

func newTokenForge(t *testing.T, clock *forgeClock) *tokenForge {
	t.Helper()

	held := &tokenForge{t: t, clock: clock, repository: http.NotFoundHandler()}
	routes := http.NewServeMux()
	routes.HandleFunc("POST /app/installations/{installation}/access_tokens", held.grant)
	routes.HandleFunc("/", held.authenticated)
	held.server = forge(t, routes.ServeHTTP)
	return held
}

func (held *tokenForge) answeringAsTheRepository(repository http.Handler) {
	held.mutex.Lock()
	defer held.mutex.Unlock()
	held.repository = repository
}

func (held *tokenForge) authenticated(writer http.ResponseWriter, request *http.Request) {
	held.mutex.Lock()
	minted, repository := "Bearer "+grantedToken(held.exchanges), held.repository
	held.mutex.Unlock()

	if held.exchangeCount() == 0 || request.Header.Get("Authorization") != minted {
		held.t.Errorf("%s %s did not present the minted installation token", request.Method, request.URL.Path)
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	repository.ServeHTTP(writer, request)
}

func (held *tokenForge) refuseFromNowOn() {
	held.mutex.Lock()
	defer held.mutex.Unlock()
	held.isRefusing = true
}

func (held *tokenForge) isRefusingNow() bool {
	held.mutex.Lock()
	defer held.mutex.Unlock()
	return held.isRefusing
}

func (held *tokenForge) grant(writer http.ResponseWriter, request *http.Request) {
	if held.isRefusingNow() {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if request.PathValue("installation") != strconv.Itoa(forgeInstallationID) {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	now := held.clock.Now()
	if !isPresentedAppJWT(held.t, request, now) || !isLeastPrivilegeAsked(held.t, request) {
		writer.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	writer.WriteHeader(http.StatusCreated)
	write(held.t, writer, recordedGrant(grantedToken(held.exchanged()), now.Add(grantedLifetime)))
}

func (held *tokenForge) exchanged() int {
	held.mutex.Lock()
	defer held.mutex.Unlock()
	held.exchanges++
	return held.exchanges
}

func (held *tokenForge) exchangeCount() int {
	held.mutex.Lock()
	defer held.mutex.Unlock()
	return held.exchanges
}

func tokenEndpoint(t *testing.T, now time.Time) *httptest.Server {
	t.Helper()

	return newTokenForge(t, clockAt(now)).server
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

	return installationTokensOn(t, server, clockAt(now))
}

func installationTokensOn(t *testing.T, server *httptest.Server, clock *forgeClock) *github.InstallationTokens {
	t.Helper()

	tokens, err := github.ParseInstallationTokens(settingsAt(server), github.Installation{
		Key:   forgeAppKey(t),
		ID:    forgeInstallationID,
		Clock: clock.Now,
	})
	if err != nil {
		t.Fatalf("parse installation tokens: %v", err)
	}
	return tokens
}

func TestParseInstallationTokensRefusesWhatCannotMintAToken(t *testing.T) {
	refusals := map[string]struct {
		spoil func(*github.Settings, *github.Installation)
		want  error
	}{
		"no App ID": {
			func(s *github.Settings, _ *github.Installation) { s.AppID = 0 }, github.ErrMalformedAppID,
		},
		"a negative installation ID": {
			func(_ *github.Settings, i *github.Installation) { i.ID = -forgeInstallationID },
			github.ErrMalformedInstallationID,
		},
		"no installation ID": {
			func(_ *github.Settings, i *github.Installation) { i.ID = 0 }, github.ErrMalformedInstallationID,
		},
		"no App key": {
			func(_ *github.Settings, i *github.Installation) { i.Key = github.AppKey{} }, github.ErrNoAppKey,
		},
		"no clock": {
			func(_ *github.Settings, i *github.Installation) { i.Clock = nil }, github.ErrNoClock,
		},
	}

	for name, refusal := range refusals {
		t.Run(name, func(t *testing.T) {
			settings := wellFormedSettings()
			installation := github.Installation{
				Key: forgeAppKey(t), ID: forgeInstallationID, Clock: func() time.Time { return mintedAt },
			}
			refusal.spoil(&settings, &installation)

			_, err := github.ParseInstallationTokens(settings, installation)

			if !errors.Is(err, refusal.want) {
				t.Fatalf("error is %v, want %v", err, refusal.want)
			}
		})
	}
}

func TestATokenEndpointErrorFailsTheExchange(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity,
		http.StatusInternalServerError, http.StatusServiceUnavailable,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			tokens := installationTokensAt(t, answering(t, status, `{"message": "refused"}`), mintedAt)

			authorization, err := github.AuthorizationOf(context.Background(), tokens)

			if !errors.Is(err, github.ErrUnexpectedStatus) || authorization != "" {
				t.Fatalf("error is %v and an authorization was presented, want %v and none",
					err, github.ErrUnexpectedStatus)
			}
		})
	}
}

func TestAGrantWithNoTokenOrNoExpiryIsRefused(t *testing.T) {
	expiry := mintedAt.Add(time.Hour).Format(time.RFC3339)
	incomplete := map[string]string{
		"no token":     `{"expires_at": "` + expiry + `"}`,
		"empty token":  `{"token": "", "expires_at": "` + expiry + `"}`,
		"no expiry":    `{"token": "` + recordedInstallationToken + `"}`,
		"empty object": `{}`,
	}

	for name, body := range incomplete {
		t.Run(name, func(t *testing.T) {
			tokens := installationTokensAt(t, answering(t, http.StatusCreated, body), mintedAt)

			_, err := github.AuthorizationOf(context.Background(), tokens)

			if !errors.Is(err, github.ErrIncompleteGrant) {
				t.Fatalf("error is %v, want %v", err, github.ErrIncompleteGrant)
			}
		})
	}
}

func TestAGrantAlreadyPastItsExpiryIsNeverPresented(t *testing.T) {
	expired := recordedGrant(recordedInstallationToken, mintedAt.Add(-time.Second))
	tokens := installationTokensAt(t, answering(t, http.StatusCreated, expired), mintedAt)

	authorization, err := github.AuthorizationOf(context.Background(), tokens)

	if !errors.Is(err, github.ErrExpiredGrant) || authorization != "" {
		t.Fatalf("error is %v and an authorization was presented, want %v and none",
			err, github.ErrExpiredGrant)
	}
}

func TestAnUnknownInstallationIsReportedAsSuchNotAsAMissingPullRequest(t *testing.T) {
	tokens := installationTokensAt(t, answering(t, http.StatusNotFound, `{"message": "Not Found"}`), mintedAt)

	_, err := github.AuthorizationOf(context.Background(), tokens)

	if !errors.Is(err, github.ErrNoSuchInstallation) || errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v and not %v",
			err, github.ErrNoSuchInstallation, domain.ErrNoSuchPullRequest)
	}
}

type exchangedOnce struct {
	clock    *forgeClock
	endpoint *tokenForge
	tokens   *github.InstallationTokens
	first    string
}

func tokensExchangedOnce(t *testing.T) exchangedOnce {
	t.Helper()

	clock := clockAt(mintedAt)
	endpoint := newTokenForge(t, clock)
	tokens := installationTokensOn(t, endpoint.server, clock)
	first, err := github.AuthorizationOf(context.Background(), tokens)
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}
	return exchangedOnce{clock: clock, endpoint: endpoint, tokens: tokens, first: first}
}

func TestACallWellBeforeExpiryReusesTheCachedToken(t *testing.T) {
	held := tokensExchangedOnce(t)
	held.clock.advance(grantedLifetime / 2)

	second, err := github.AuthorizationOf(context.Background(), held.tokens)

	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if second != held.first || held.endpoint.exchangeCount() != 1 {
		t.Errorf("half an hour before expiry the token was exchanged again: %d exchanges",
			held.endpoint.exchangeCount())
	}
}

func TestConcurrentCallsShareOneExchange(t *testing.T) {
	const callers = 16
	clock := clockAt(mintedAt)
	endpoint := newTokenForge(t, clock)
	tokens := installationTokensOn(t, endpoint.server, clock)
	var everyone sync.WaitGroup

	for range callers {
		everyone.Go(func() {
			if _, err := github.AuthorizationOf(context.Background(), tokens); err != nil {
				t.Errorf("concurrent call: %v", err)
			}
		})
	}
	everyone.Wait()

	if endpoint.exchangeCount() != 1 {
		t.Errorf("%d concurrent callers caused %d exchanges, want 1", callers, endpoint.exchangeCount())
	}
}

func TestACallNearExpiryGetsAFreshToken(t *testing.T) {
	held := tokensExchangedOnce(t)
	held.clock.advance(grantedLifetime - time.Minute)

	authorization, err := github.AuthorizationOf(context.Background(), held.tokens)

	if err != nil {
		t.Fatalf("call near expiry: %v", err)
	}
	if authorization != "Bearer "+grantedToken(2) {
		t.Errorf("a minute before expiry the old token was presented: %d exchanges",
			held.endpoint.exchangeCount())
	}
}

func TestAFailedRefreshFailsTheCallAndNeverPresentsTheExpiringToken(t *testing.T) {
	held := tokensExchangedOnce(t)
	held.endpoint.refuseFromNowOn()
	held.clock.advance(grantedLifetime - time.Minute)

	authorization, err := github.AuthorizationOf(context.Background(), held.tokens)

	if !errors.Is(err, github.ErrUnexpectedStatus) || authorization != "" {
		t.Fatalf("error is %v and an authorization was presented, want %v and none",
			err, github.ErrUnexpectedStatus)
	}
}

func TestAnInstallationTokenIsExchangedForTheAppJWTAskingExactlyLeastPrivilege(t *testing.T) {
	tokens := installationTokensAt(t, tokenEndpoint(t, mintedAt), mintedAt)

	authorization, err := github.AuthorizationOf(context.Background(), tokens)

	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if authorization != "Bearer "+grantedToken(1) {
		t.Errorf("the presented authorization is not the exchanged installation token")
	}
}
