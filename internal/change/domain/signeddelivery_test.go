// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	deliveredBody = `{"action":"opened","number":412}`
	appSecret     = "s3cret"
	trueSignature = "sha256=51e90d780f2a25ff6328cb24733ed9c4b05b70202696d272ab38890b3f016416"
)

func TestParseSignedDeliveryCarriesABodySignedWithTheAppSecret(t *testing.T) {
	delivered, err := domain.ParseSignedDelivery(
		[]byte(deliveredBody), trueSignature, appSecret)

	if err != nil {
		t.Fatalf("parse a correctly signed delivery: %v", err)
	}
	if string(delivered.Body()) != deliveredBody {
		t.Errorf("body is %q, want %q", delivered.Body(), deliveredBody)
	}
}

func TestParseSignedDeliveryRefusesADeliveryItCannotAuthenticate(t *testing.T) {
	otherBody := `{"action":"closed","number":412}`
	refused := map[string]struct {
		body, signature string
		want            error
	}{
		"no signature at all":        {deliveredBody, "", domain.ErrUnsignedDelivery},
		"no algorithm prefix":        {deliveredBody, strings.TrimPrefix(trueSignature, "sha256="), domain.ErrUnsignedDelivery},
		"the legacy sha1 header":     {deliveredBody, "sha1=" + strings.TrimPrefix(trueSignature, "sha256="), domain.ErrUnsignedDelivery},
		"not hexadecimal":            {deliveredBody, "sha256=notahexdigest", domain.ErrUnsignedDelivery},
		"truncated to an odd length": {deliveredBody, trueSignature[:20], domain.ErrUnsignedDelivery},
		"truncated but still hex":    {deliveredBody, trueSignature[:21], domain.ErrMissignedDelivery},
		"a digest of another body":   {otherBody, trueSignature, domain.ErrMissignedDelivery},
		"one flipped hex digit":      {deliveredBody, "sha256=61e90d780f2a25ff6328cb24733ed9c4b05b70202696d272ab38890b3f016416", domain.ErrMissignedDelivery},
	}

	for form, delivery := range refused {
		t.Run(form, func(t *testing.T) {
			delivered, err := domain.ParseSignedDelivery(
				[]byte(delivery.body), delivery.signature, appSecret)

			if !errors.Is(err, delivery.want) {
				t.Fatalf("error is %v, want %v", err, delivery.want)
			}
			if len(delivered.Body()) != 0 {
				t.Fatalf("a refused delivery carried a body: %q", delivered.Body())
			}
		})
	}
}
