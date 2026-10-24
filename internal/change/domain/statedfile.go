// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type StatedFile struct {
	File          File
	PreviousPath  string
	Patch         string
	IsPatchStated bool
}
