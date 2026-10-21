// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

var mintedAt = time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)

type appJWTClaims struct {
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func TestAnAppJWTIsSignedRS256AndNamesTheAppForUnderTenMinutes(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	minted, err := github.AppJWTAt(key, forgeAppID, mintedAt)

	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	claims := verifiedClaimsOf(t, minted, &key.PublicKey)
	if claims.Issuer != "4242" {
		t.Errorf("iss is %q, want the App ID 4242", claims.Issuer)
	}
	if claims.IssuedAt != mintedAt.Add(-time.Minute).Unix() {
		t.Errorf("iat is %d, want %d, backdated sixty seconds",
			claims.IssuedAt, mintedAt.Add(-time.Minute).Unix())
	}
	if claims.ExpiresAt <= mintedAt.Unix() || claims.ExpiresAt >= mintedAt.Add(10*time.Minute).Unix() {
		t.Errorf("exp is %d, want after %d and under ten minutes later",
			claims.ExpiresAt, mintedAt.Unix())
	}
}

func verifiedClaimsOf(t *testing.T, minted string, public *rsa.PublicKey) appJWTClaims {
	t.Helper()

	parts := strings.Split(minted, ".")
	if len(parts) != 3 {
		t.Fatalf("the JWT has %d parts, want 3", len(parts))
	}
	if header := decodedSegment(t, parts[0]); header != `{"alg":"RS256","typ":"JWT"}` {
		t.Errorf("header is %s", header)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	signature := []byte(decodedSegment(t, parts[2]))
	if err := rsa.VerifyPKCS1v15(public, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatalf("the signature does not verify against the App's public key: %v", err)
	}
	var claims appJWTClaims
	if err := json.Unmarshal([]byte(decodedSegment(t, parts[1])), &claims); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	return claims
}

func decodedSegment(t *testing.T, segment string) string {
	t.Helper()

	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		t.Fatalf("decode %q: %v", segment, err)
	}
	return string(decoded)
}
