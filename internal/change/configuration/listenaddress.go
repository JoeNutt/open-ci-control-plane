// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"net/netip"
)

const unchosenPort = 0

var errOffLoopback = errors.New("the address is not a loopback address")

var errUnchosenPort = errors.New("the address leaves its port to the operating system")

func listenAddressIn(stated string) (netip.AddrPort, error) {
	if stated == "" {
		return netip.AddrPort{}, listenAddress.missing()
	}
	parsed, err := netip.ParseAddrPort(stated)
	if err != nil {
		return netip.AddrPort{}, listenAddress.invalid(err)
	}
	if !parsed.Addr().IsLoopback() {
		return netip.AddrPort{}, listenAddress.invalid(errOffLoopback)
	}
	if parsed.Port() == unchosenPort {
		return netip.AddrPort{}, listenAddress.invalid(errUnchosenPort)
	}
	return parsed, nil
}
