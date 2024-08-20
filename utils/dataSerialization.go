package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
)

// ? SerializeArray serializes an array of key-value pairs to a Base64-encoded string
func SerializeStream(data map[string][]string) (string, error) {

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)

	if err != nil {
		return "", fmt.Errorf("serialization error: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return encoded, nil
}

// ? DeserializeArray deserializes a Base64-encoded string to an array of key-value pairs
func DeserializeStream(encoded string) (map[string][]string, error) {
	serialized, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding error: %v", err)
	}

	var data map[string][]string
	buf := bytes.NewBuffer(serialized)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&data)

	if err != nil {
		return nil, fmt.Errorf("deserialization error: %v", err)
	}

	return data, nil
}
