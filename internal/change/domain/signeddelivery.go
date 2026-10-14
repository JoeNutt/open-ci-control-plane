// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const signaturePrefix = "sha256="

var ErrUnsignedDelivery = errors.New("delivery carries no usable signature")

var ErrMissignedDelivery = errors.New("delivery signature does not verify")

type SignedDelivery struct {
	body []byte
}

func (delivered SignedDelivery) Body() []byte { return delivered.body }

func ParseSignedDelivery(body []byte, signature, secret string) (SignedDelivery, error) {
	claimed, found := strings.CutPrefix(signature, signaturePrefix)
	if !found {
		return SignedDelivery{}, ErrUnsignedDelivery
	}
	offered, err := hex.DecodeString(claimed)
	if err != nil {
		return SignedDelivery{}, ErrUnsignedDelivery
	}
	signing := hmac.New(sha256.New, []byte(secret))
	signing.Write(body)
	if !hmac.Equal(offered, signing.Sum(nil)) {
		return SignedDelivery{}, ErrMissignedDelivery
	}
	return SignedDelivery{body: body}, nil
}
