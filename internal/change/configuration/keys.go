// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

var errNotAnObject = errors.New("the configuration is not one JSON object")

var errUnknownKey = errors.New("the key is not exactly the name of an item")

var errRepeatedKey = errors.New("the key is given twice")

var errTrailingContent = errors.New("content follows the configuration object")

var errNullValue = errors.New("the item is stated as null")

var jsonNull = []byte("null")

type statedItems map[item]json.RawMessage

type itemValue interface {
	string | int64 | []int64 | []string
}

func statedItemsIn(content []byte) (statedItems, error) {
	walking := json.NewDecoder(bytes.NewReader(content))
	opening, err := walking.Token()
	if err != nil {
		return nil, err
	}
	if opening != json.Delim('{') {
		return nil, errNotAnObject
	}
	stated := statedItems{}
	for walking.More() {
		named, value, err := statedItemAt(walking, stated)
		if err != nil {
			return nil, err
		}
		stated[named] = value
	}
	return stated, endOfObjectIn(walking)
}

func statedItemAt(walking *json.Decoder, stated statedItems) (item, json.RawMessage, error) {
	token, err := walking.Token()
	if err != nil {
		return "", nil, err
	}
	name, isName := token.(string)
	if !isName || !slices.Contains(items, item(name)) {
		return "", nil, fmt.Errorf("%w: %v", errUnknownKey, token)
	}
	if _, isRepeated := stated[item(name)]; isRepeated {
		return "", nil, fmt.Errorf("%w: %q", errRepeatedKey, name)
	}
	var value json.RawMessage
	if err := walking.Decode(&value); err != nil {
		return "", nil, err
	}
	return item(name), value, nil
}

func endOfObjectIn(walking *json.Decoder) error {
	if _, err := walking.Token(); err != nil {
		return err
	}
	if _, err := walking.Token(); !errors.Is(err, io.EOF) {
		return errors.Join(errTrailingContent, err)
	}
	return nil
}

func valueOf[Value itemValue](stated statedItems, named item) (Value, error) {
	var value Value
	raw, isStated := stated[named]
	if !isStated {
		return value, named.missing()
	}
	if bytes.Equal(bytes.TrimSpace(raw), jsonNull) {
		return value, named.invalid(errNullValue)
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, named.invalid(err)
	}
	return value, nil
}
