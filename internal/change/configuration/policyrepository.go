// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

func policyRepositoryIn(stated statedItems) (domain.PolicyRepository, error) {
	named, err := valueOf[string](stated, policyRepositoryItem)
	if err != nil {
		return domain.PolicyRepository{}, err
	}
	repository, err := domain.ParsePolicyRepository(named)
	if err != nil {
		return domain.PolicyRepository{}, policyRepositoryItem.invalid(err)
	}
	return repository, nil
}

func policyInstallationIDIn(stated statedItems) (int64, error) {
	installation, err := valueOf[int64](stated, policyInstallationIDItem)
	if err != nil {
		return 0, err
	}
	if installation < firstGitHubID {
		return 0, policyInstallationIDItem.invalid(errNonPositiveID)
	}
	return installation, nil
}
