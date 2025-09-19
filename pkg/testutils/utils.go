package testutils

import (
	"encoding/json"
	"errors"
	"strings"
)

func ValueOfJsonField(jsonAsString string, field string) (string, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonAsString))

	var decodedData map[string]interface{}
	err := jsonDecoder.Decode(&decodedData)
	if err != nil {
		return "", err
	}
	value, containsField := decodedData[field]
	if !containsField {
		return "", nil
	}
	s, ok := value.(string)
	if !ok {
		return "", errors.New("value is not a string")
	}
	return s, nil
}

func ValueOfJsonFieldInt(jsonAsString string, field string) (int, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonAsString))
	jsonDecoder.UseNumber()

	var decodedData map[string]interface{}
	err := jsonDecoder.Decode(&decodedData)
	if err != nil {
		return 0, err
	}
	value, containsField := decodedData[field]
	if !containsField {
		return 0, nil
	}

	number, ok := value.(json.Number)
	if !ok {
		return 0, errors.New("value is not a json number")
	}
	valueInt64, err := number.Int64()
	if err != nil {
		return 0, err
	}

	return int(valueInt64), nil
}

func ContainsJsonField(jsonAsString string, field string) bool {
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonAsString))
	jsonDecoder.UseNumber()

	var decodedData map[string]interface{}
	err := jsonDecoder.Decode(&decodedData)
	if err != nil {
		return false
	}

	_, containsField := decodedData[field]
	return containsField
}
