// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

const MaxAppKeyBytes = 16 << 10

var ErrAppKeyTooLarge = errors.New("the App key file is larger than the App key byte bound")

var ErrMalformedAppKey = errors.New("the App key is not one PEM-encoded PKCS#1 RSA private key")

const pkcs1PrivateKeyBlock = "RSA PRIVATE KEY"

type AppKey struct {
	key *rsa.PrivateKey
}

func ParseAppKey(encoded []byte) (AppKey, error) {
	if len(encoded) > MaxAppKeyBytes {
		return AppKey{}, fmt.Errorf("%w: %d bytes, more than %d",
			ErrAppKeyTooLarge, len(encoded), MaxAppKeyBytes)
	}
	block, rest := pem.Decode(encoded)
	if !isOnePKCS1Block(block, rest) {
		return AppKey{}, ErrMalformedAppKey
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return AppKey{}, fmt.Errorf("%w: %w", ErrMalformedAppKey, err)
	}
	return AppKey{key: key}, nil
}

func isOnePKCS1Block(block *pem.Block, rest []byte) bool {
	return block != nil && block.Type == pkcs1PrivateKeyBlock && len(rest) == 0
}
