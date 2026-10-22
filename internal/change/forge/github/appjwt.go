// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const (
	appJWTBackdate = time.Minute
	appJWTAhead    = 9 * time.Minute
)

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

var rs256Header = jwtHeader{Algorithm: "RS256", Type: "JWT"}

type appJWTClaims struct {
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func appJWTAt(appKey AppKey, app appID, now time.Time) (string, error) {
	signingInput, err := appJWTSigningInput(app, now)
	if err != nil {
		return "", err
	}
	signature, err := rs256SignatureOf(appKey.key, signingInput)
	if err != nil {
		return "", err
	}
	return signingInput + "." + signature, nil
}

func appJWTSigningInput(app appID, now time.Time) (string, error) {
	header, err := encodedSegment(rs256Header)
	if err != nil {
		return "", err
	}
	claims, err := encodedSegment(appJWTClaims{
		Issuer:    app.String(),
		IssuedAt:  now.Add(-appJWTBackdate).Unix(),
		ExpiresAt: now.Add(appJWTAhead).Unix(),
	})
	if err != nil {
		return "", err
	}
	return header + "." + claims, nil
}

func encodedSegment[Segment jwtHeader | appJWTClaims](segment Segment) (string, error) {
	encoded, err := json.Marshal(segment)
	if err != nil {
		return "", fmt.Errorf("encode an App JWT segment: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func rs256SignatureOf(key *rsa.PrivateKey, signingInput string) (string, error) {
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign the App JWT: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(signature), nil
}
