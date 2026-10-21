// SPDX-License-Identifier: GPL-3.0-or-later

package ingest_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

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

func TestActionsTokensReadTheRepositoryOfAVerifiedToken(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)

	repository, err := tokens.RepositoryOf(t.Context(), issuer.sign(t, issuer.claims(governedRepository)))

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
			_, err := issuer.tokensFor(t, instanceURL).RepositoryOf(t.Context(), unverifiable)

			if !errors.Is(err, ingest.ErrUnverifiedToken) {
				t.Fatalf("want ErrUnverifiedToken, got %v", err)
			}
		})
	}
}

func TestActionsTokensFetchTheKeySetOnceForKeysTheyHold(t *testing.T) {
	issuer := newIssuer(t)
	tokens := issuer.tokensFor(t, instanceURL)

	for range 2 {
		if _, err := tokens.RepositoryOf(t.Context(), issuer.sign(t, issuer.claims(governedRepository))); err != nil {
			t.Fatalf("verify a token GitHub Actions issued: %v", err)
		}
	}

	if fetches := issuer.keySetFetches.Load(); fetches != 1 {
		t.Fatalf("fetched the key set %d times, want 1", fetches)
	}
}

type issuer struct {
	t             *testing.T
	server        *httptest.Server
	signingKey    jwk.Key
	keySetFetches atomic.Int64
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	KeySets string `json:"jwks_uri"`
}

func newIssuer(t *testing.T) *issuer {
	t.Helper()

	minted := &issuer{t: t, signingKey: newSigningKey(t, signingKeyID)}
	minted.server = httptest.NewTLSServer(http.HandlerFunc(minted.serve))
	t.Cleanup(minted.server.Close)
	return minted
}

func (minted *issuer) serve(answer http.ResponseWriter, asked *http.Request) {
	switch asked.URL.Path {
	case discoveryPath:
		minted.write(answer, minted.discovery())
	case keySetPath:
		minted.keySetFetches.Add(1)
		minted.write(answer, minted.publishedKeys())
	default:
		answer.WriteHeader(http.StatusNotFound)
	}
}

func (minted *issuer) discovery() []byte {
	encoded, err := json.Marshal(discoveryDocument{Issuer: minted.server.URL, KeySets: minted.server.URL + keySetPath})
	if err != nil {
		minted.t.Errorf("encode discovery: %v", err)
	}
	return encoded
}

func (minted *issuer) publishedKeys() []byte {
	public, err := jwk.PublicKeyOf(minted.signingKey)
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

func (minted *issuer) tokensFor(t *testing.T, audience string) *ingest.ActionsTokens {
	t.Helper()

	tokens, err := ingest.ParseActionsTokens(ingest.ActionsTokenSettings{
		Issuer:   minted.server.URL,
		Audience: audience,
		Client:   minted.server.Client(),
		Now:      func() time.Time { return issuedAt },
	})
	if err != nil {
		t.Fatalf("configure token verification: %v", err)
	}
	return tokens
}

func (minted *issuer) claims(repository string) *jwt.Builder {
	return jwt.NewBuilder().
		Issuer(minted.server.URL).
		Audience([]string{instanceURL}).
		NotBefore(issuedAt.Add(-time.Minute)).
		Expiration(issuedAt.Add(tokenLifetime)).
		Claim(repositoryClaim, repository)
}

func (minted *issuer) sign(t *testing.T, claims *jwt.Builder) string {
	t.Helper()

	return signed(t, minted.built(t, claims), minted.signingKey)
}

func (minted *issuer) signWithout(t *testing.T, claims *jwt.Builder, absent string) string {
	t.Helper()

	token := minted.built(t, claims)
	if err := token.Remove(absent); err != nil {
		t.Fatalf("remove %s: %v", absent, err)
	}
	return signed(t, token, minted.signingKey)
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
