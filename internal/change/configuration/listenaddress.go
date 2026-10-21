// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"net/netip"
)

func listenAddressPresentIn(stated string) error {
	if stated == "" {
		return listenAddress.missing()
	}
	parsed, err := netip.ParseAddrPort(stated)
	if err != nil {
		return listenAddress.invalid(err)
	}
	if !parsed.Addr().IsLoopback() || parsed.Port() == 0 {
		return listenAddress.invalid(errors.New("not loopback"))
	}
	return nil
}
