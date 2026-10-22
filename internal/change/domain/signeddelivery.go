// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

const signaturePrefix = "sha256="

var ErrUnsignedDelivery = errors.New("delivery carries no usable signature")

var ErrMissignedDelivery = errors.New("delivery signature does not verify")

type WebhookSecret string

func (secret WebhookSecret) String() string { return redacted }

func (secret WebhookSecret) GoString() string { return strconv.Quote(redacted) }

type SignedDelivery struct {
	body []byte
}

func (delivered SignedDelivery) Body() []byte { return delivered.body }

func ParseSignedDelivery(
	body []byte, signature string, secret WebhookSecret,
) (SignedDelivery, error) {
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
