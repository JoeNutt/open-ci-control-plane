// SPDX-License-Identifier: GPL-3.0-or-later

package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	MaxIssuerDocumentBytes = 64 << 10
	discoveryPath          = "/.well-known/openid-configuration"
	keySetRefetchInterval  = time.Minute
)

var ErrIssuerKeysUnavailable = errors.New("the token issuer's signing keys could not be read")

var errOversizedIssuerDocument = fmt.Errorf("the issuer's document exceeds %d bytes", MaxIssuerDocumentBytes)

type issuerKeys struct {
	issuer      domain.HTTPSOrigin
	client      *http.Client
	now         func() time.Time
	guard       sync.Mutex
	held        jwk.Set
	refetchedAt time.Time
}

type discoveryDocument struct {
	KeySets string `json:"jwks_uri"`
}

func (keys *issuerKeys) holding(ctx context.Context, keyID string) (jwk.Set, error) {
	keys.guard.Lock()
	defer keys.guard.Unlock()

	if keys.isHolding(keyID) || keys.isRefetchRecent() {
		return keys.held, nil
	}
	return keys.replaced(ctx)
}

func (keys *issuerKeys) isHolding(keyID string) bool {
	if keys.held == nil {
		return false
	}
	_, isHeld := keys.held.LookupKeyID(keyID)
	return isHeld
}

func (keys *issuerKeys) isRefetchRecent() bool {
	return !keys.refetchedAt.IsZero() && keys.now().Sub(keys.refetchedAt) < keySetRefetchInterval
}

func (keys *issuerKeys) replaced(ctx context.Context) (jwk.Set, error) {
	if keys.held != nil {
		keys.refetchedAt = keys.now()
	}
	published, err := keys.published(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIssuerKeysUnavailable, err)
	}
	keys.held = published
	return published, nil
}

func (keys *issuerKeys) published(ctx context.Context) (jwk.Set, error) {
	location, err := keys.keySetLocation(ctx)
	if err != nil {
		return nil, err
	}
	document, err := keys.documentAt(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("read the issuer's key set: %w", err)
	}
	published, err := jwk.Parse(document)
	if err != nil {
		return nil, fmt.Errorf("parse the issuer's key set: %w", err)
	}
	return published, nil
}

func (keys *issuerKeys) keySetLocation(ctx context.Context) (string, error) {
	document, err := keys.documentAt(ctx, keys.issuer.String()+discoveryPath)
	if err != nil {
		return "", fmt.Errorf("read the issuer's configuration: %w", err)
	}
	var discovered discoveryDocument
	if err := json.Unmarshal(document, &discovered); err != nil {
		return "", fmt.Errorf("decode the issuer's configuration: %w", err)
	}
	return discovered.KeySets, nil
}

func (keys *issuerKeys) documentAt(ctx context.Context, location string) ([]byte, error) {
	asked, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	answer, err := keys.client.Do(asked)
	if err != nil {
		return nil, err
	}
	return boundedBodyOf(answer)
}

func boundedBodyOf(answer *http.Response) ([]byte, error) {
	body, readFailure := io.ReadAll(io.LimitReader(answer.Body, MaxIssuerDocumentBytes+1))
	if err := errors.Join(readFailure, answer.Body.Close()); err != nil {
		return nil, err
	}
	if answer.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the issuer answered %d", answer.StatusCode)
	}
	if len(body) > MaxIssuerDocumentBytes {
		return nil, errOversizedIssuerDocument
	}
	return body, nil
}
