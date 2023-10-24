package zmtp

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Metadata []byte

// Properties returns a slice of properties held in this metadata.
func (m Metadata) Properties(f func(name string, value string)) error {
	idx := 0

	for idx < len(m) {
		nameLen := int(m[idx])
		idx += 1
		if idx+nameLen >= len(m) {
			return ErrNameTooLong
		}
		name := string(m[idx : idx+nameLen])
		idx += nameLen

		if idx+4 >= len(m) {
			return fmt.Errorf("%w: not enough bytes for next value name", ErrInvalidMetadata)
		}
		valueLen := int(binary.BigEndian.Uint32(m[idx:]))
		idx += 4

		if idx+valueLen > len(m) {
			return fmt.Errorf("%w: not enough bytes for next value", ErrInvalidMetadata)
		}
		value := string(m[idx : idx+valueLen])
		idx += valueLen

		f(name, value)
	}

	return nil
}

// AddProperty to the Metadata.
func (m *Metadata) AddProperty(name string, value string) error {
	buffer := bytes.NewBuffer(*m)
	if len(name) > 255 {
		return ErrNameTooLong
	}

	if err := buffer.WriteByte(byte(len(name))); err != nil {
		return err
	}

	if _, err := buffer.WriteString(name); err != nil {
		return err
	}

	if err := binary.Write(buffer, binary.BigEndian, uint32(len(value))); err != nil {
		return err
	}

	_, err := buffer.WriteString(value)
	*m = buffer.Bytes()
	return err
}

type invalidMetadata struct{}

func (invalidMetadata) Error() string {
	return "Invalid metadata"
}

// ErrInvalidMetadata is returned when the metadata cannot be parsed into its properties.
var ErrInvalidMetadata invalidMetadata

type nameTooLong struct{}

func (nameTooLong) Error() string {
	return "Property name > 255 bytes"
}

// ErrNameTooLong is returns when the metadata contains a name specifier which is too long for the metadata length.
var ErrNameTooLong nameTooLong
