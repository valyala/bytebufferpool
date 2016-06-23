package bytebufferpool

import "sync"

const (
	minBitSize = 8
	steps      = 20

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)
)

type byteBufferPool struct {
	// Pools are segemented into power-of-two sized buffers
	// from minSize bytes to maxSize.
	//
	// This allows reducing fragmentation of ByteBuffer objects.
	pools [steps]sync.Pool
}

func (p *byteBufferPool) Acquire() *ByteBuffer {
	pools := &p.pools
	for i := 0; i < steps; i++ {
		v := pools[i].Get()
		if v != nil {
			return v.(*ByteBuffer)
		}
	}

	return &ByteBuffer{
		B: make([]byte, 0, minSize),
	}
}

func (p *byteBufferPool) Release(b *ByteBuffer) {
	n := cap(b.B)
	if n > maxSize {
		// Oversized buffer.
		// Drop it.
		return
	}
	if (n >> 2) > len(b.B) {
		// Under-used buffer capacity.
		// Drop it.
		return
	}

	b.B = b.B[:0]
	idx := bitSize(n-1) >> minBitSize
	p.pools[idx].Put(b)
}

func bitSize(n int) int {
	s := 0
	for n > 0 {
		n >>= 1
		s++
	}
	return s
}
