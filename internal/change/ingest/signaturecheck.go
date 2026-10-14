// SPDX-License-Identifier: GPL-3.0-or-later

// Package ingest accepts forge events, admitting only deliveries it can authenticate.
package ingest

import (
	"bytes"
	"io"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const signatureHeader = "X-Hub-Signature-256"

type RejectionCounter interface {
	CountRejectedDelivery()
}

type SignatureCheck struct {
	Accepted   http.Handler
	Secret     domain.WebhookSecret
	Rejections RejectionCounter
}

func (checking SignatureCheck) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	body, err := io.ReadAll(asked.Body)
	if err != nil {
		answer.WriteHeader(http.StatusBadRequest)
		return
	}
	delivered, err := domain.ParseSignedDelivery(
		body, asked.Header.Get(signatureHeader), checking.Secret)
	if err != nil {
		checking.Rejections.CountRejectedDelivery()
		answer.WriteHeader(http.StatusUnauthorized)
		return
	}
	checking.Accepted.ServeHTTP(answer, carrying(asked, delivered))
}

func carrying(asked *http.Request, delivered domain.SignedDelivery) *http.Request {
	verified := asked.Clone(asked.Context())
	verified.Body = io.NopCloser(bytes.NewReader(delivered.Body()))
	verified.ContentLength = int64(len(delivered.Body()))
	return verified
}
