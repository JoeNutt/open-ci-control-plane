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

func ParseAppKey(b []byte) (AppKey, error) {
	blk, _ := pem.Decode(b)
	k, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	return AppKey{k}, err
}
