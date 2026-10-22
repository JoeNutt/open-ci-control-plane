// SPDX-License-Identifier: GPL-3.0-or-later

package ingest_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const (
	governedRepository = "JoeNutt/open-ci-control-plane"
	instanceURL        = "https://control-plane.example"
	signingKeyID       = "actions-signing-key"
	rsaKeyBits         = 2048
	discoveryPath      = "/.well-known/openid-configuration"
	keySetPath         = "/.well-known/jwks"
	tokenLifetime      = 5 * time.Minute
	repositoryClaim    = "repository"
)

var issuedAt = time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)

func TestParseActionsTokensRefusesIncompleteSettings(t *testing.T) {
	complete := ingest.ActionsTokenSettings{
		Issuer: originOf(t, "https://token.actions.githubusercontent.com"), Audience: originOf(t, instanceURL),
		Client: &http.Client{Timeout: time.Second}, Now: time.Now}
	for form, without := range map[string]func(ingest.ActionsTokenSettings) ingest.ActionsTokenSettings{
		"no issuer": func(settings ingest.ActionsTokenSettings) ingest.ActionsTokenSettings {
			settings.Issuer = domain.HTTPSOrigin{}
			return settings
		},
		"no audience": func(settings ingest.ActionsTokenSettings) ingest.ActionsTokenSettings {
			settings.Audience = domain.HTTPSOrigin{}
			return settings
		},
		"no client": func(settings ingest.ActionsTokenSettings) ingest.ActionsTokenSettings {
			settings.Client = nil
			return settings
		},
		"no timeout on the client": func(settings ingest.ActionsTokenSettings) ingest.ActionsTokenSettings {
			settings.Client = &http.Client{}
			return settings
		},
		"a negative timeout on the client": func(settings ingest.ActionsTokenSettings) ingest.ActionsTokenSettings {
			settings.Client = &http.Client{Timeout: -time.Second}
			return settings
		},
		"no clock": func(settings ingest.ActionsTokenSettings) ingest.ActionsTokenSettings {
			settings.Now = nil
			return settings
		},
	} {
		t.Run(form, func(t *testing.T) {
			_, err := ingest.ParseActionsTokens(without(complete))

			if !errors.Is(err, ingest.ErrIncompleteActionsTokens) {
				t.Fatalf("want ErrIncompleteActionsTokens, got %v", err)
			}
		})
	}
}

func TestActionsTokensReadTheRepositoryOfAVerifiedToken(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)

	repository, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t))

	if err != nil {
		t.Fatalf("verify a token GitHub Actions issued: %v", err)
	}
	if repository.String() != governedRepository {
		t.Fatalf("repository is %s, want %s", repository, governedRepository)
	}
}

func TestActionsTokensRefuseATokenTheyCannotVerify(t *testing.T) {
	issuer := newIssuer(t)
	for name, unverifiable := range map[string]string{
		"another issuer":   issuer.sign(t, issuer.claims(governedRepository).Issuer("https://token.actions.example")),
		"another audience": issuer.sign(t, issuer.claims(governedRepository).Audience([]string{"https://another-instance.example"})),
		"expired":          issuer.sign(t, issuer.claims(governedRepository).Expiration(issuedAt.Add(-time.Second))),
		"not yet valid":    issuer.sign(t, issuer.claims(governedRepository).NotBefore(issuedAt.Add(time.Second))),
		"no expiry":        issuer.signWithout(t, issuer.claims(governedRepository), jwt.ExpirationKey),
		"no not-before":    issuer.signWithout(t, issuer.claims(governedRepository), jwt.NotBeforeKey),
		"unpublished key":  issuer.forge(t, issuer.claims(governedRepository), newSigningKey(t, signingKeyID)),
		"unsigned":         issuer.unsigned(t, issuer.claims(governedRepository)),
		"no repository":    issuer.signWithout(t, issuer.claims(governedRepository), repositoryClaim),
		"not a repository": issuer.sign(t, issuer.claims("../../etc/passwd")),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := issuer.tokensFor(t, instanceURL).RepositoryOf(t.Context(), identity(t, unverifiable))

			if !errors.Is(err, domain.ErrUnverifiedIdentityToken) {
				t.Fatalf("want ErrUnverifiedIdentityToken, got %v", err)
			}
			if strings.Contains(err.Error(), unverifiable) {
				t.Fatalf("the refusal carries the token: %v", err)
			}
		})
	}
}

func TestActionsTokensRefuseATokenOverTheBoundBeforeParsingIt(t *testing.T) {
	issuer := newIssuer(t)
	padding := strings.Repeat("a", ingest.MaxActionsTokenBytes)
	oversized := issuer.sign(t, issuer.claims(governedRepository).Claim("padding", padding))

	_, err := issuer.tokensFor(t, instanceURL).RepositoryOf(t.Context(), identity(t, oversized))

	if !errors.Is(err, domain.ErrUnverifiedIdentityToken) {
		t.Fatalf("want ErrUnverifiedIdentityToken, got %v", err)
	}
	if fetches := issuer.keySetFetches.Load(); fetches != 0 {
		t.Fatalf("fetched the key set %d times for an oversized token, want 0", fetches)
	}
}

func TestActionsTokensRefuseAnIssuerTheyCannotRead(t *testing.T) {
	notJSON := []byte("<html>")
	for name, misanswering := range map[string]func(*issuer){
		"configuration over the bound": func(minted *issuer) {
			minted.answer(discoveryPath, http.StatusOK, padded(minted.discovery()))
		},
		"configuration refused": func(minted *issuer) {
			minted.answer(discoveryPath, http.StatusServiceUnavailable, minted.discovery())
		},
		"configuration not JSON": func(minted *issuer) { minted.answer(discoveryPath, http.StatusOK, notJSON) },
		"key set over the bound": func(minted *issuer) {
			minted.answer(keySetPath, http.StatusOK, padded(minted.publishedKeys()))
		},
		"key set refused": func(minted *issuer) {
			minted.answer(keySetPath, http.StatusServiceUnavailable, minted.publishedKeys())
		},
		"key set not a key set": func(minted *issuer) { minted.answer(keySetPath, http.StatusOK, notJSON) },
	} {
		t.Run(name, func(t *testing.T) {
			issuer := newIssuer(t)
			misanswering(issuer)

			_, err := issuer.tokensFor(t, instanceURL).RepositoryOf(t.Context(), issuer.governedToken(t))

			if !errors.Is(err, domain.ErrIdentityIssuerUnavailable) {
				t.Fatalf("want ErrIdentityIssuerUnavailable, got %v", err)
			}
		})
	}
}

func TestActionsTokensFetchTheKeySetOnceForKeysTheyHold(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)

	for range 2 {
		if _, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t)); err != nil {
			t.Fatalf("verify a token GitHub Actions issued: %v", err)
		}
	}

	if fetches := issuer.keySetFetches.Load(); fetches != 1 {
		t.Fatalf("fetched the key set %d times, want 1", fetches)
	}
}

func TestActionsTokensRefetchTheKeySetForAKeyTheyDoNotHold(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)
	if _, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t)); err != nil {
		t.Fatalf("verify a token before rotation: %v", err)
	}

	issuer.rotate(t, "rotated-signing-key")
	_, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t))

	if err != nil {
		t.Fatalf("verify a token signed by the rotated key: %v", err)
	}
}

func TestActionsTokensRefetchTheKeySetAtMostOnceForUnknownKeys(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)
	if _, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t)); err != nil {
		t.Fatalf("verify a token GitHub Actions issued: %v", err)
	}

	for _, unknownKeyID := range []string{"unknown-1", "unknown-2", "unknown-3"} {
		forged := issuer.forge(t, issuer.claims(governedRepository), newSigningKey(t, unknownKeyID))
		if _, err := tokens.RepositoryOf(t.Context(), identity(t, forged)); !errors.Is(err, domain.ErrUnverifiedIdentityToken) {
			t.Fatalf("want ErrUnverifiedIdentityToken, got %v", err)
		}
	}

	if fetches := issuer.keySetFetches.Load(); fetches != 2 {
		t.Fatalf("fetched the key set %d times, want 2", fetches)
	}
}

func TestActionsTokensRefetchTheKeySetAgainOnceADayHasPassed(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)
	if _, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t)); err != nil {
		t.Fatalf("verify a token GitHub Actions issued: %v", err)
	}
	forged := issuer.forge(t, issuer.claims(governedRepository), newSigningKey(t, "unknown"))
	if _, err := tokens.RepositoryOf(t.Context(), identity(t, forged)); !errors.Is(err, domain.ErrUnverifiedIdentityToken) {
		t.Fatalf("want ErrUnverifiedIdentityToken, got %v", err)
	}

	issuer.advance(24 * time.Hour)
	issuer.rotate(t, "rotated-signing-key")
	_, err := tokens.RepositoryOf(t.Context(), issuer.governedToken(t))

	if err != nil {
		t.Fatalf("verify a token signed by a key rotated a day later: %v", err)
	}
}

type issuer struct {
	t             *testing.T
	server        *httptest.Server
	guard         sync.Mutex
	signingKey    jwk.Key
	keySetFetches atomic.Int64
	instant       time.Time
	misanswers    map[string]misanswer
}

type misanswer struct {
	status int
	body   []byte
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	KeySets string `json:"jwks_uri"`
}

func newIssuer(t *testing.T) *issuer {
	t.Helper()

	minted := &issuer{
		t: t, signingKey: newSigningKey(t, signingKeyID), instant: issuedAt, misanswers: map[string]misanswer{}}
	minted.server = httptest.NewTLSServer(http.HandlerFunc(minted.serve))
	t.Cleanup(minted.server.Close)
	return minted
}

func (minted *issuer) rotate(t *testing.T, keyID string) {
	t.Helper()

	rotated := newSigningKey(t, keyID)
	minted.guard.Lock()
	defer minted.guard.Unlock()
	minted.signingKey = rotated
}

func (minted *issuer) answer(path string, status int, body []byte) {
	minted.guard.Lock()
	defer minted.guard.Unlock()
	minted.misanswers[path] = misanswer{status: status, body: body}
}

func padded(document []byte) []byte {
	padding := `,"padding":"` + strings.Repeat("a", ingest.MaxIssuerDocumentBytes) + `"}`
	return append(bytes.TrimSuffix(document, []byte("}")), padding...)
}

func (minted *issuer) advance(elapsed time.Duration) {
	minted.guard.Lock()
	defer minted.guard.Unlock()
	minted.instant = minted.instant.Add(elapsed)
}

func (minted *issuer) now() time.Time {
	minted.guard.Lock()
	defer minted.guard.Unlock()
	return minted.instant
}

func (minted *issuer) currentKey() jwk.Key {
	minted.guard.Lock()
	defer minted.guard.Unlock()
	return minted.signingKey
}

func (minted *issuer) serve(answer http.ResponseWriter, asked *http.Request) {
	if asked.URL.Path == keySetPath {
		minted.keySetFetches.Add(1)
	}
	if wrong, isMisanswered := minted.misanswerFor(asked.URL.Path); isMisanswered {
		answer.WriteHeader(wrong.status)
		minted.write(answer, wrong.body)
		return
	}
	switch asked.URL.Path {
	case discoveryPath:
		minted.write(answer, minted.discovery())
	case keySetPath:
		minted.write(answer, minted.publishedKeys())
	default:
		answer.WriteHeader(http.StatusNotFound)
	}
}

func (minted *issuer) misanswerFor(path string) (misanswer, bool) {
	minted.guard.Lock()
	defer minted.guard.Unlock()
	wrong, isMisanswered := minted.misanswers[path]
	return wrong, isMisanswered
}

func (minted *issuer) discovery() []byte {
	encoded, err := json.Marshal(discoveryDocument{Issuer: minted.server.URL, KeySets: minted.server.URL + keySetPath})
	if err != nil {
		minted.t.Errorf("encode discovery: %v", err)
	}
	return encoded
}

func (minted *issuer) publishedKeys() []byte {
	public, err := jwk.PublicKeyOf(minted.currentKey())
	if err != nil {
		minted.t.Errorf("derive public key: %v", err)
	}
	published := jwk.NewSet()
	if err := published.AddKey(public); err != nil {
		minted.t.Errorf("publish key: %v", err)
	}
	encoded, err := json.Marshal(published)
	if err != nil {
		minted.t.Errorf("encode key set: %v", err)
	}
	return encoded
}

func (minted *issuer) write(answer http.ResponseWriter, body []byte) {
	answer.Header().Set("Content-Type", "application/json")
	if _, err := answer.Write(body); err != nil {
		minted.t.Errorf("serve %d bytes: %v", len(body), err)
	}
}

func (minted *issuer) tokensFor(t *testing.T, audience string) domain.CallerRepositories {
	t.Helper()

	tokens, err := ingest.ParseActionsTokens(ingest.ActionsTokenSettings{
		Issuer:   originOf(t, minted.server.URL),
		Audience: originOf(t, audience),
		Client:   minted.timedClient(),
		Now:      minted.now,
	})
	if err != nil {
		t.Fatalf("configure token verification: %v", err)
	}
	return tokens
}

func (minted *issuer) timedClient() *http.Client {
	client := minted.server.Client()
	client.Timeout = 5 * time.Second
	return client
}

func (minted *issuer) claims(repository string) *jwt.Builder {
	return jwt.NewBuilder().
		Issuer(minted.server.URL).
		Audience([]string{instanceURL}).
		NotBefore(minted.now().Add(-time.Minute)).
		Expiration(minted.now().Add(tokenLifetime)).
		Claim(repositoryClaim, repository)
}

func (minted *issuer) sign(t *testing.T, claims *jwt.Builder) string {
	t.Helper()

	return signed(t, minted.built(t, claims), minted.currentKey())
}

func (minted *issuer) signWithout(t *testing.T, claims *jwt.Builder, absent string) string {
	t.Helper()

	token := minted.built(t, claims)
	if err := token.Remove(absent); err != nil {
		t.Fatalf("remove %s: %v", absent, err)
	}
	return signed(t, token, minted.currentKey())
}

func (minted *issuer) built(t *testing.T, claims *jwt.Builder) jwt.Token {
	t.Helper()

	token, err := claims.Build()
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	return token
}

func (minted *issuer) forge(t *testing.T, claims *jwt.Builder, unpublished jwk.Key) string {
	t.Helper()

	return signed(t, minted.built(t, claims), unpublished)
}

func (minted *issuer) unsigned(t *testing.T, claims *jwt.Builder) string {
	t.Helper()

	signed, err := jwt.Sign(minted.built(t, claims), jwt.WithInsecureNoSignature())
	if err != nil {
		t.Fatalf("leave token unsigned: %v", err)
	}
	return string(signed)
}

func signed(t *testing.T, token jwt.Token, key jwk.Key) string {
	t.Helper()

	signature, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), key))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return string(signature)
}

func (minted *issuer) governedToken(t *testing.T) domain.IdentityToken {
	t.Helper()

	return identity(t, minted.sign(t, minted.claims(governedRepository)))
}

func identity(t *testing.T, presented string) domain.IdentityToken {
	t.Helper()

	token, err := domain.ParseIdentityToken(presented)
	if err != nil {
		t.Fatalf("parse identity token: %v", err)
	}
	return token
}

func newSigningKey(t *testing.T, keyID string) jwk.Key {
	t.Helper()

	private, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key, err := jwk.Import(private)
	if err != nil {
		t.Fatalf("import key: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, keyID); err != nil {
		t.Fatalf("name key: %v", err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		t.Fatalf("assign algorithm: %v", err)
	}
	return key
}

func originOf(t *testing.T, stated string) domain.HTTPSOrigin {
	t.Helper()
	origin, err := domain.ParseHTTPSOrigin(stated)
	if err != nil {
		t.Fatalf("parse origin %q: %v", stated, err)
	}
	return origin
}
