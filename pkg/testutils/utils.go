package testutils

import (
	"encoding/json"
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
	return value.(string), nil
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

	valueInt64, err := value.(json.Number).Int64()
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
