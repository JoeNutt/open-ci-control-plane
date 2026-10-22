// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"net/netip"
)

const unchosenPort = 0

var errPublicAddress = errors.New("the address is neither loopback nor private")

var errUnchosenPort = errors.New("the address leaves its port to the operating system")

func listenAddressIn(stated statedItems) (netip.AddrPort, error) {
	text, err := valueOf[string](stated, listenAddressItem)
	if err != nil {
		return netip.AddrPort{}, err
	}
	address, err := netip.ParseAddrPort(text)
	if err != nil {
		return netip.AddrPort{}, listenAddressItem.invalid(err)
	}
	if !isUnreachableFromOutside(address.Addr()) {
		return netip.AddrPort{}, listenAddressItem.invalid(errPublicAddress)
	}
	if address.Port() == unchosenPort {
		return netip.AddrPort{}, listenAddressItem.invalid(errUnchosenPort)
	}
	return address, nil
}

func isUnreachableFromOutside(address netip.Addr) bool {
	return address.IsLoopback() || address.IsPrivate()
}
