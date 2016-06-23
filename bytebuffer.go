package bytebufferpool

// ByteBuffer provides byte buffer, which can be used for minimizing
// memory allocations.
//
// ByteBuffer may be used with functions appending data to the given []byte
// slice. See example code for details.
//
// Use Acquire for obtaining an empty byte buffer.
type ByteBuffer struct {

	// B is a byte buffer to use in append-like workloads.
	// See example code for details.
	B []byte
}

// Write implements io.Writer - it appends p to ByteBuffer.B
func (b *ByteBuffer) Write(p []byte) (int, error) {
	b.B = append(b.B, p...)
	return len(p), nil
}

// WriteString appends s to ByteBuffer.B
func (b *ByteBuffer) WriteString(s string) (int, error) {
	b.B = append(b.B, s...)
	return len(s), nil
}

// Set sets ByteBuffer.B to p
func (b *ByteBuffer) Set(p []byte) {
	b.B = append(b.B[:0], p...)
}

// SetString sets ByteBuffer.B to s
func (b *ByteBuffer) SetString(s string) {
	b.B = append(b.B[:0], s...)
}

// Reset makes ByteBuffer.B empty.
func (b *ByteBuffer) Reset() {
	b.B = b.B[:0]
}

// Acquire returns an empty byte buffer from the pool.
//
// Acquired byte buffer may be returned to the pool via Release call.
// This reduces the number of memory allocations required for byte buffer
// management.
func Acquire() *ByteBuffer {
	return defaultByteBufferPool.Acquire()
}

// Release returns byte buffer to the pool.
//
// ByteBuffer.B mustn't be touched after returning it to the pool.
// Otherwise data races will occur.
func Release(b *ByteBuffer) {
	defaultByteBufferPool.Release(b)
}

var defaultByteBufferPool byteBufferPool
