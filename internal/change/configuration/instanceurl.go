// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

func instanceURLIn(stated statedItems) (domain.HTTPSOrigin, error) {
	text, err := valueOf[string](stated, instanceURLItem)
	if err != nil {
		return domain.HTTPSOrigin{}, err
	}
	origin, err := domain.ParseHTTPSOrigin(text)
	if err != nil {
		return domain.HTTPSOrigin{}, instanceURLItem.invalid(err)
	}
	return origin, nil
}
