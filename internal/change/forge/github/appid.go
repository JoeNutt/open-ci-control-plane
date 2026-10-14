// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrMalformedAppID = errors.New("the forge settings carry no positive GitHub App id")

type appID int64

func parseAppID(stated int64) (appID, error) {
	if stated <= 0 {
		return 0, fmt.Errorf("%w: %d", ErrMalformedAppID, stated)
	}
	return appID(stated), nil
}

func (app appID) String() string {
	return strconv.FormatInt(int64(app), 10)
}
