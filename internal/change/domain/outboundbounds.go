// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "time"

const OutboundAttemptLimit = 5

const OutboundRetryWindow = 5 * time.Minute

const OutboundCallTimeout = 10 * time.Second
