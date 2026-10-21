// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func listenAddressPresentIn(stated string) error {
	if stated == "" {
		return listenAddress.missing()
	}
	return nil
}
