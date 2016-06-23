package bytebufferpool

import (
	"sort"
	"sync"
	"sync/atomic"
)

const (
	minBitSize = 8
	steps      = 20

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)

	calibrateCallsThreshold = 42000
)

type byteBufferPool struct {
	idxs        [steps]uint64
	calls       [steps]uint64
	calibrating uint64

	// Pools are segemented into power-of-two sized buffers
	// from minSize bytes to maxSize.
	//
	// This allows reducing fragmentation of ByteBuffer objects.
	pools [steps]sync.Pool
}

func (p *byteBufferPool) Acquire() *ByteBuffer {
	for _, idx := range p.idxs {
		v := p.pools[idx].Get()
		if v != nil {
			return v.(*ByteBuffer)
		}
	}

	return &ByteBuffer{
		B: make([]byte, 0, minSize),
	}
}

func (p *byteBufferPool) Release(b *ByteBuffer) {
	bCap := cap(b.B)
	if bCap > maxSize {
		// Oversized buffer.
		// Drop it.
		return
	}

	idx := bitSize(bCap-1) >> minBitSize
	b.B = b.B[:0]
	p.pools[idx].Put(b)

	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}
}

func (p *byteBufferPool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	var a callidxs
	for i := uint64(0); i < steps; i++ {
		a = append(a, callidx{
			calls: atomic.SwapUint64(&p.calls[i], 0),
			idx:   i,
		})
	}
	sort.Sort(a)

	for i := 0; i < steps; i++ {
		atomic.StoreUint64(&p.idxs[i], a[i].idx)
	}

	atomic.StoreUint64(&p.calibrating, 0)
}

type callidx struct {
	calls uint64
	idx   uint64
}

type callidxs []callidx

func (ci callidxs) Len() int {
	return len(ci)
}

func (ci callidxs) Less(i, j int) bool {
	return ci[i].calls > ci[j].calls
}

func (ci callidxs) Swap(i, j int) {
	ci[i], ci[j] = ci[j], ci[i]
}

func bitSize(n int) int {
	s := 0
	for n > 0 {
		n >>= 1
		s++
	}
	return s
}
