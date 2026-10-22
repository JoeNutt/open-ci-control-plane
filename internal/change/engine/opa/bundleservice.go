// SPDX-License-Identifier: GPL-3.0-or-later

package opa

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
)

const loopbackScheme = "http"

var ErrMalformedBundleService = errors.New("the bundle service is not an http loopback address with a port")

type BundleService struct {
	value string
}

func ParseBundleService(stated string) (BundleService, error) {
	address, err := url.Parse(stated)
	if err != nil {
		return BundleService{}, fmt.Errorf("%w: %w", ErrMalformedBundleService, err)
	}
	if !isBareLoopbackService(address) {
		return BundleService{}, ErrMalformedBundleService
	}
	return BundleService{value: stated}, nil
}

func isBareLoopbackService(address *url.URL) bool {
	host, err := netip.ParseAddrPort(address.Host)
	return err == nil && host.Addr().IsLoopback() && host.Port() != 0 &&
		address.Scheme == loopbackScheme && address.User == nil &&
		address.Path == "" && address.RawQuery == "" && !address.ForceQuery && address.Fragment == ""
}

func (service BundleService) String() string { return service.value }
