// SPDX-License-Identifier: GPL-3.0-or-later

package opa

func BundlePollingOf(settings Settings) (minDelaySeconds, maxDelaySeconds int64) {
	polling := configurationFor(settings).Bundles[bundleNameFor(settings.Family)].Polling
	return polling.MinDelaySeconds, polling.MaxDelaySeconds
}
