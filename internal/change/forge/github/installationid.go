// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrMalformedInstallationID = errors.New("the forge settings carry no positive App installation id")

type installationID int64

func parseInstallationID(stated int64) (installationID, error) {
	if stated <= 0 {
		return 0, fmt.Errorf("%w: %d", ErrMalformedInstallationID, stated)
	}
	return installationID(stated), nil
}

func (installation installationID) String() string {
	return strconv.FormatInt(int64(installation), 10)
}
