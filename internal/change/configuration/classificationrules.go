// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func classificationRulesIn(stated statedItems) ([]byte, error) {
	return fileContentOf(stated, classificationRulesFileItem, MaxFileBytes)
}

func ClassificationRulesParsedBy[Rules any](configured Configuration, parse func([]byte) (Rules, error)) (Rules, error) {
	return parsedFileItem(classificationRulesFileItem, configured.classificationRules, parse)
}

func parsedFileItem[Value any](named item, content []byte, parse func([]byte) (Value, error)) (Value, error) {
	parsed, err := parse(content)
	if err != nil {
		var none Value
		return none, named.invalid(err)
	}
	return parsed, nil
}
