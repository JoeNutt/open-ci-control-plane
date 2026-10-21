// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

type AppKey struct {
	key *rsa.PrivateKey
}

func ParseAppKey(encoded []byte) (AppKey, error) {
	block, _ := pem.Decode(encoded)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	return AppKey{key: key}, err
}
