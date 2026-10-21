// SPDX-License-Identifier: GPL-3.0-or-later

package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

const MaxIssuerDocumentBytes = 64 << 10

var ErrIssuerKeysUnavailable = errors.New("the token issuer's signing keys could not be read")

const (
	discoveryPath         = "/.well-known/openid-configuration"
	maxDiscoveryBytes     = MaxIssuerDocumentBytes
	keySetRefetchInterval = time.Minute
)

var errOversizedConfiguration = fmt.Errorf("the issuer's configuration exceeds %d bytes", maxDiscoveryBytes)

type issuerKeys struct {
	issuer      *url.URL
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
		return nil, err
	}
	keys.held = published
	return published, nil
}

func (keys *issuerKeys) published(ctx context.Context) (jwk.Set, error) {
	location, err := keys.keySetLocation(ctx)
	if err != nil {
		return nil, err
	}
	asked, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch the issuer's key set: %w", err)
	}
	answer, err := keys.client.Do(asked)
	if err != nil {
		return nil, fmt.Errorf("fetch the issuer's key set: %w", err)
	}
	document, err := boundedBodyOf(answer)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIssuerKeysUnavailable, err)
	}
	published, err := jwk.Parse(document)
	if err != nil {
		return nil, fmt.Errorf("parse the issuer's key set: %w", err)
	}
	return published, nil
}

func (keys *issuerKeys) keySetLocation(ctx context.Context) (string, error) {
	asked, err := http.NewRequestWithContext(ctx, http.MethodGet, keys.issuer.JoinPath(discoveryPath).String(), nil)
	if err != nil {
		return "", fmt.Errorf("ask the issuer for its configuration: %w", err)
	}
	answer, err := keys.client.Do(asked)
	if err != nil {
		return "", fmt.Errorf("ask the issuer for its configuration: %w", err)
	}
	document, err := boundedBodyOf(answer)
	if err != nil {
		return "", fmt.Errorf("read the issuer's configuration: %w", err)
	}
	var discovered discoveryDocument
	if err := json.Unmarshal(document, &discovered); err != nil {
		return "", fmt.Errorf("decode the issuer's configuration: %w", err)
	}
	return discovered.KeySets, nil
}

func boundedBodyOf(answer *http.Response) ([]byte, error) {
	body, readFailure := io.ReadAll(io.LimitReader(answer.Body, maxDiscoveryBytes+1))
	if err := errors.Join(readFailure, answer.Body.Close()); err != nil {
		return nil, err
	}
	if answer.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the issuer answered %d", answer.StatusCode)
	}
	if len(body) > maxDiscoveryBytes {
		return nil, errOversizedConfiguration
	}
	return body, nil
}
