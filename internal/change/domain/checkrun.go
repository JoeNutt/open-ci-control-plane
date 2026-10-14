// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type CheckRun struct {
	Name       string
	Conclusion Conclusion
	Title      string
	Summary    string
	Text       string
}
