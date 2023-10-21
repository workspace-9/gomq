package null

type ByteReader byte

func (b ByteReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	p[0] = byte(b)
	return 1, nil
}
