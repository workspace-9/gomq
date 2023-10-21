package zmtp

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Command struct {
	Name string
	Body []byte
}

// WriteTo writes a command to the given writer.
func (c Command) WriteTo(w io.Writer) (int64, error) {
	total := int64(0)
	if bodyLen := c.BodyLen(); bodyLen <= 255 {
		n, err := w.Write([]byte{0x04, uint8(bodyLen)})
		total += int64(n)
		if err != nil {
			return total, err
		}
	} else {
		data := [9]byte{}
		data[0] = 0x06
		binary.BigEndian.PutUint64(data[1:], uint64(bodyLen))
		n, err := w.Write(data[:])
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	n, err := w.Write([]byte{uint8(len(c.Name))})
	total += int64(n)
	if err != nil {
		return total, err
	}

	n, err = w.Write([]byte(c.Name))
	total += int64(n)
	if err != nil {
		return total, err
	}

	n, err = w.Write(c.Body)
	total += int64(n)
	return total, err
}

type invalidCommandSize struct{}

func (invalidCommandSize) Error() string {
	return "Invalid command size"
}

var ErrInvalidCommandSize invalidCommandSize

type invalidNameLength struct{}

func (invalidNameLength) Error() string {
	return "Invalid name length"
}

var ErrInvalidNameLength invalidNameLength

// ReadFrom reads a command from the given reader.
func (c *Command) ReadFrom(r io.Reader) (int64, error) {
	total := int64(0)
	var b [1]byte
	n, err := io.ReadFull(r, b[:])
	total += int64(n)
	if err != nil {
		return total, err
	}

	var cmdLen uint64
	switch b[0] {
	case 0x04:
		n, err := io.ReadFull(r, b[:])
		total += int64(n)
		if err != nil {
			return total, err
		}

		cmdLen = uint64(b[0])
	case 0x06:
		if err := binary.Read(r, binary.BigEndian, &cmdLen); err != nil {
			return total, err
		}
		total += 8
	default:
		return total, fmt.Errorf("%w: unrecognized size specified %x", ErrInvalidCommandSize, b[0])
	}

	body := make([]byte, cmdLen)
	n, err = io.ReadFull(r, body)
	total += int64(n)
	if err != nil {
		return total, err
	}

	nameLen := int(body[0])
	if nameLen > len(body[1:]) {
		return total, fmt.Errorf("%w: name length > body size", ErrInvalidNameLength)
	}
	c.Name = string(body[1 : nameLen+1])
	c.Body = body[nameLen+1:]
	return total, nil
}

// BodyLen returns the body length of the given message.
func (c Command) BodyLen() int32 {
	return int32(len(c.Name)) + 1 + int32(len(c.Body))
}
