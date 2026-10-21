// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

var errNotAnObject = errors.New("the configuration is not one JSON object")

var errUnknownKey = errors.New("the key is not exactly the name of an item")

var errRepeatedKey = errors.New("the key is given twice")

func keysCheckedIn(content []byte) error {
	walking := json.NewDecoder(bytes.NewReader(content))
	opening, err := walking.Token()
	if err != nil {
		return err
	}
	if opening != json.Delim('{') {
		return errNotAnObject
	}
	var seen []item
	for walking.More() {
		key, err := keyAt(walking, seen)
		if err != nil {
			return err
		}
		seen = append(seen, key)
	}
	return nil
}

func keyAt(walking *json.Decoder, seen []item) (item, error) {
	token, err := walking.Token()
	if err != nil {
		return "", err
	}
	name, isName := token.(string)
	if !isName || !slices.Contains(items, item(name)) {
		return "", fmt.Errorf("%w: %v", errUnknownKey, token)
	}
	if slices.Contains(seen, item(name)) {
		return "", fmt.Errorf("%w: %q", errRepeatedKey, name)
	}
	var value json.RawMessage
	if err := walking.Decode(&value); err != nil {
		return "", err
	}
	return item(name), nil
}
