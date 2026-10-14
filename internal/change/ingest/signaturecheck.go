// SPDX-License-Identifier: GPL-3.0-or-later

// Package ingest accepts forge events, admitting only deliveries it can authenticate.
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

type RejectionCounter interface {
	CountRejectedDelivery()
}

type Settings struct {
	Accepted   http.Handler
	Secret     domain.WebhookSecret
	Rejections RejectionCounter
}

var ErrIncompleteSignatureCheck = errors.New("signature check is missing a part it cannot work without")

type SignatureCheck struct {
	accepted   http.Handler
	secret     domain.WebhookSecret
	rejections RejectionCounter
}

func ParseSignatureCheck(settings Settings) (SignatureCheck, error) {
	absent := []func(Settings) string{
		noHandlerWaiting,
		noSecretToVerifyAgainst,
		noCounterForRejections,
	}
	for _, missing := range absent {
		if part := missing(settings); part != "" {
			return SignatureCheck{}, fmt.Errorf("%w: %s", ErrIncompleteSignatureCheck, part)
		}
	}
	return SignatureCheck{
		accepted:   settings.Accepted,
		secret:     settings.Secret,
		rejections: settings.Rejections,
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

func noCounterForRejections(settings Settings) string {
	if settings.Rejections == nil {
		return "nothing is counting refusals"
	}
	return ""
}

func (checking SignatureCheck) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	body, err := io.ReadAll(asked.Body)
	if err != nil {
		checking.rejections.CountRejectedDelivery()
		answer.WriteHeader(http.StatusBadRequest)
		return
	}
	delivered, err := domain.ParseSignedDelivery(
		body, asked.Header.Get(signatureHeader), checking.secret)
	if err != nil {
		checking.rejections.CountRejectedDelivery()
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
