package conn

import "io"

type buffer struct {
	bs  []byte
	len uint64
	n   uint64
}

// ReadN will read n bytes from an io.Reader
// Note: Internal byteslice resets on each read
func (b *buffer) ReadN(r io.Reader, n uint64) (err error) {
	var rn int
	b.n = 0

	if n > b.len {
		// Our internal slice is too small, grow before reading
		b.bs = make([]byte, n)
		b.len = n
	}

	for b.n < n && err == nil {
		rn, err = r.Read(b.bs[b.n:n])
		b.n += uint64(rn)
	}

	if err == io.EOF && b.n == n {
		err = nil
	}

	return
}

func (b *buffer) Bytes() []byte {
	return b.bs[:b.n]
}
