package proto

import (
	"errors"
	"fmt"
)

func appendString(data []byte, s string) ([]byte, error) {
	length := len(s)

	const maxLength = (1 << 8) - 1
	if length > maxLength {
		return nil, fmt.Errorf("string is too long. Max: %d. Got: %d", maxLength, length)
	}

	data = append(data, uint8(length))
	sBytes := []byte(s)
	data = append(data, sBytes[:length]...)

	return data, nil
}

func readString(data []byte) ([]byte, string, error) {
	if len(data) < 1 {
		return nil, "", errors.New("could not read a string from an empty byte slice")
	}

	length := int(data[0])
	data = data[1:]
	if len(data) < length {
		return nil, "", fmt.Errorf(
			"slice only has %d bytes, but string is said to be %d bytes long",
			len(data),
			length,
		)
	}

	s := string(data[:length])

	return data[length:], s, nil
}
