package zmtp

import (
	"encoding/binary"
	"io"
)

// Message is a zmtp message.
type Message struct {
	More bool
	Body []byte
}

// WriteTo writes a message to the given writer.
func (m Message) WriteTo(w io.Writer) (int64, error) {
	total := int64(0)

	if l := len(m.Body); l <= 255 {
		var data [2]byte
		if m.More {
			data[0] = 0x01
		} else {
			data[0] = 0x00
		}
		data[1] = byte(l)
		n, err := w.Write(data[:])
		total += int64(n)
		if err != nil {
			return total, err
		}
	} else {
		var data [9]byte
		if m.More {
			data[0] = 0x03
		} else {
			data[0] = 0x02
		}

		binary.BigEndian.PutUint64(data[1:], uint64(l))
		n, err := w.Write(data[:])
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	n, err := w.Write(m.Body)
	total += int64(n)
	return total, err
}

// ReadFrom reads a message from the given reader.
func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	total := int64(0)
	var buf [1]byte
	n, err := io.ReadFull(r, buf[:])
	total += int64(n)
	if err != nil {
		return total, err
	}

	var messageLen uint64
	switch buf[0] {
	case 0x01, 0x00:
		m.More = buf[0] == 0x01
		n, err := io.ReadFull(r, buf[:])
		total += int64(n)
		if err != nil {
			return total, err
		}
		messageLen = uint64(buf[0])
	case 0x02, 0x03:
		m.More = buf[0] == 0x03
		err := binary.Read(r, binary.BigEndian, &messageLen)
		if err != nil {
			return total, err
		}
		total += 8
	}

	m.Body = make([]byte, messageLen)
	n, err = io.ReadFull(r, m.Body)
	total += int64(n)
	return total, err
}
