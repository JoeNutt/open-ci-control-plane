// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

var ErrMalformedAppKey = errors.New("the App key is not one PEM-encoded PKCS#1 RSA private key")

type AppKey struct {
	key *rsa.PrivateKey
}

func ParseAppKey(encoded []byte) (AppKey, error) {
	block, rest := pem.Decode(encoded)
	if block == nil || block.Type != "RSA PRIVATE KEY" || len(rest) != 0 {
		return AppKey{}, ErrMalformedAppKey
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return AppKey{}, fmt.Errorf("%w: %w", ErrMalformedAppKey, err)
	}
	return AppKey{key: key}, nil
}
