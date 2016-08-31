package bytebufferpool

import "io"

// ByteBuffer provides byte buffer, which can be used for minimizing
// memory allocations.
//
// ByteBuffer may be used with functions appending data to the given []byte
// slice. See example code for details.
//
// Use Get for obtaining an empty byte buffer.
type ByteBuffer struct {

	// B is a byte buffer to use in append-like workloads.
	// See example code for details.
	buf []byte
}

// NewByteBuffer creates and initializes a new ByteBuffer using buf as its initial
// contents.
func NewByteBuffer(buf []byte) *ByteBuffer { return &ByteBuffer{buf: buf} }

// Len returns the size of the byte buffer.
func (b *ByteBuffer) Len() int {
	return len(b.buf)
}

// Cap returns the capacity of the buffer's underlying byte slice.
func (b *ByteBuffer) Cap() int {
	return cap(b.buf)
}

// ReadFrom implements io.ReaderFrom.
//
// The function appends all the data read from r to b.
func (b *ByteBuffer) ReadFrom(r io.Reader) (int64, error) {
	p := b.buf
	nStart := int64(len(p))
	nMax := int64(cap(p))
	n := nStart
	if nMax == 0 {
		nMax = 64
		p = make([]byte, nMax)
	} else {
		p = p[:nMax]
	}
	for {
		if n == nMax {
			nMax *= 2
			bNew := make([]byte, nMax)
			copy(bNew, p)
			p = bNew
		}
		nn, err := r.Read(p[n:])
		n += int64(nn)
		if err != nil {
			b.buf = p[:n]
			n -= nStart
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
	}
}

// WriteTo implements io.WriterTo.
func (b *ByteBuffer) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b.buf)
	return int64(n), err
}

// Bytes returns b.B, i.e. all the bytes accumulated in the buffer.
//
// The purpose of this function is bytes.Buffer compatibility.
func (b *ByteBuffer) Bytes() []byte {
	return b.buf
}

// Write implements io.Writer - it appends p to ByteBuffer.B
func (b *ByteBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// WriteByte appends the byte c to the buffer.
//
// The purpose of this function is bytes.Buffer compatibility.
//
// The function always returns nil.
func (b *ByteBuffer) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}

// WriteString appends s to ByteBuffer.B.
func (b *ByteBuffer) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}

// Set sets ByteBuffer.B to p.
func (b *ByteBuffer) Set(p []byte) {
	b.buf = append(b.buf[:0], p...)
}

// SetString sets ByteBuffer.B to s.
func (b *ByteBuffer) SetString(s string) {
	b.buf = append(b.buf[:0], s...)
}

// String returns string representation of ByteBuffer.B.
func (b *ByteBuffer) String() string {
	return string(b.buf)
}

// Reset makes ByteBuffer.B empty.
func (b *ByteBuffer) Reset() {
	b.buf = b.buf[:0]
}
