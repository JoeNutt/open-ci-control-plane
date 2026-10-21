// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

func appJWTAt(key *rsa.PrivateKey, app appID, now time.Time) (string, error) {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	c := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"iss":"4242","iat":%d,"exp":%d}`, now.Unix()-60, now.Unix()+540)))
	d := sha256.Sum256([]byte(h + "." + c))
	s, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, d[:])
	if err != nil {
		return "", err
	}
	return h + "." + c + "." + base64.RawURLEncoding.EncodeToString(s), nil
}
