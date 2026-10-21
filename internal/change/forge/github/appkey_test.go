// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"sync"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

var forgeRSAKey = sync.OnceValue(func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
})

func pemOf(blockType string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}

func forgeAppKeyPEM() []byte {
	return pemOf(github.PKCS1PrivateKeyBlock, x509.MarshalPKCS1PrivateKey(forgeRSAKey()))
}

func forgeAppKey(t *testing.T) github.AppKey {
	t.Helper()

	key, err := github.ParseAppKey(forgeAppKeyPEM())
	if err != nil {
		t.Fatalf("parse the App key: %v", err)
	}
	return key
}

func TestParseAppKeyRefusesWhatIsNotAPKCS1RSAPrivateKey(t *testing.T) {
	pkcs8, err := x509.MarshalPKCS8PrivateKey(forgeRSAKey())
	if err != nil {
		t.Fatalf("marshal PKCS#8: %v", err)
	}
	refused := map[string][]byte{
		"empty":                 nil,
		"not PEM":               []byte("not a key"),
		"a PKCS#8 block":        pemOf("PRIVATE KEY", pkcs8),
		"a public key":          pemOf("RSA PUBLIC KEY", x509.MarshalPKCS1PublicKey(&forgeRSAKey().PublicKey)),
		"garbage in the block":  pemOf(github.PKCS1PrivateKeyBlock, []byte("garbage")),
		"bytes after the block": append(forgeAppKeyPEM(), "trailing"...),
		"two blocks":            append(forgeAppKeyPEM(), forgeAppKeyPEM()...),
	}

	for name, encoded := range refused {
		t.Run(name, func(t *testing.T) {
			_, err := github.ParseAppKey(encoded)

			if !errors.Is(err, github.ErrMalformedAppKey) {
				t.Fatalf("error is %v, want %v", err, github.ErrMalformedAppKey)
			}
		})
	}
}
