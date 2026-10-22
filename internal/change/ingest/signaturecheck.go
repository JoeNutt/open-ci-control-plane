// SPDX-License-Identifier: GPL-3.0-or-later

package ingest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const signatureHeader = "X-Hub-Signature-256"

const MaxDeliveryBytes = 8 << 20

type RefusalRecorder interface {
	RecordRefusal(refused *http.Request, reason error)
}

type Settings struct {
	Accepted http.Handler
	Secret   domain.WebhookSecret
	Refusals RefusalRecorder
}

var ErrIncompleteSignatureCheck = errors.New("signature check is missing a part it cannot work without")

type SignatureCheck struct {
	accepted http.Handler
	secret   domain.WebhookSecret
	refusals RefusalRecorder
}

func ParseSignatureCheck(settings Settings) (SignatureCheck, error) {
	absent := []func(Settings) string{
		noHandlerWaiting,
		noSecretToVerifyAgainst,
		noRecorderForRefusals,
	}
	for _, missing := range absent {
		if part := missing(settings); part != "" {
			return SignatureCheck{}, fmt.Errorf("%w: %s", ErrIncompleteSignatureCheck, part)
		}
	}
	return SignatureCheck{
		accepted: settings.Accepted,
		secret:   settings.Secret,
		refusals: settings.Refusals,
	}, nil
}

func noHandlerWaiting(settings Settings) string {
	if settings.Accepted == nil {
		return "nothing is waiting for an admitted delivery"
	}
	return ""
}

func noSecretToVerifyAgainst(settings Settings) string {
	if settings.Secret == "" {
		return "no secret to verify a signature against"
	}
	return ""
}

func noRecorderForRefusals(settings Settings) string {
	if settings.Refusals == nil {
		return "nothing is recording refusals"
	}
	return ""
}

func (checking SignatureCheck) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(answer, asked.Body, MaxDeliveryBytes))
	if err != nil {
		checking.refusals.RecordRefusal(asked, err)
		answer.WriteHeader(statusForUnreadableBody(err))
		return
	}
	delivered, err := domain.ParseSignedDelivery(
		body, asked.Header.Get(signatureHeader), checking.secret)
	if err != nil {
		checking.refusals.RecordRefusal(asked, err)
		answer.WriteHeader(http.StatusUnauthorized)
		return
	}
	checking.accepted.ServeHTTP(answer, carrying(asked, delivered))
}

func carrying(asked *http.Request, delivered domain.SignedDelivery) *http.Request {
	verified := asked.Clone(asked.Context())
	verified.Body = io.NopCloser(bytes.NewReader(delivered.Body()))
	verified.ContentLength = int64(len(delivered.Body()))
	return verified
}

func statusForUnreadableBody(err error) int {
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}
