package conn

import (
	"encoding/binary"
	"io"
)

// lengthy is a binary helper
type lengthy struct {
	d [8]byte
}

func (l *lengthy) Write(w io.Writer, val uint64) (err error) {
	binary.LittleEndian.PutUint64(l.d[:], val)
	_, err = w.Write(l.d[:])
	return
}

func (l *lengthy) Read(r io.Reader) (val uint64, err error) {
	if _, err = io.ReadFull(r, l.d[:]); err != nil {
		return
	}

	val = binary.LittleEndian.Uint64(l.d[:])
	return
}
