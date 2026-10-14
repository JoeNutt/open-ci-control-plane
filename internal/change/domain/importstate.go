// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ImportState string

const (
	ImportStateResolved ImportState = "resolved"
	ImportStateUnknown  ImportState = "unknown"
	ImportStateErrored  ImportState = "errored"
)
