package bytebufferpool

import (
	"sort"
	"sync"
	"sync/atomic"
)

const (
	minBitSize = 6
	steps      = 20

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)

	calibrateCallsThreshold = 42000
	maxPercentile           = 0.95
)

type byteBufferPool struct {
	calibrating uint64

	defaultSize uint64
	maxSize     uint64

	calls [steps]uint64
	idxs  [steps]uint64
	pools [steps]sync.Pool
}

func (p *byteBufferPool) Acquire() *ByteBuffer {
	for i := 0; i < steps; i++ {
		idx := atomic.LoadUint64(&p.idxs[i])
		if idx >= steps {
			break
		}
		v := p.pools[idx].Get()
		if v != nil {
			return v.(*ByteBuffer)
		}
	}

	return &ByteBuffer{
		B: make([]byte, 0, atomic.LoadUint64(&p.defaultSize)),
	}
}

func (p *byteBufferPool) Release(b *ByteBuffer) {
	bLen := uint64(len(b.B))
	idx := index(bLen)
	if idx < 0 {
		idx = 0
	} else if idx >= steps {
		idx = steps - 1
	}

	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	maxSize := atomic.LoadUint64(&p.maxSize)
	bCap := uint64(cap(b.B))
	if maxSize > 0 && bCap <= maxSize {
		idx = index(bCap)
		b.B = b.B[:0]
		p.pools[idx].Put(b)
	}
}

func (p *byteBufferPool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	a := make(callSizes, 0, steps)
	var callsSum uint64
	for i := uint64(0); i < steps; i++ {
		calls := atomic.SwapUint64(&p.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,
			size:  minSize << i,
		})
	}
	sort.Sort(a)

	defaultSize := a[0].size
	maxSize := defaultSize

	maxSum := uint64(float64(callsSum) * maxPercentile)
	callsSum = 0
	for i := 0; i < steps; i++ {
		idx := uint64(steps)
		if callsSum <= maxSum {
			callsSum += a[i].calls
			size := a[i].size
			if size > maxSize {
				maxSize = size
			}
			idx = index(size)
		}
		atomic.StoreUint64(&p.idxs[i], idx)
	}

	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)

	atomic.StoreUint64(&p.calibrating, 0)
}

type callSize struct {
	calls uint64
	size  uint64
}

type callSizes []callSize

func (ci callSizes) Len() int {
	return len(ci)
}

func (ci callSizes) Less(i, j int) bool {
	return ci[i].calls > ci[j].calls
}

func (ci callSizes) Swap(i, j int) {
	ci[i], ci[j] = ci[j], ci[i]
}

func index(n uint64) uint64 {
	return bitSize(n-1) - minBitSize
}

func bitSize(n uint64) uint64 {
	s := uint64(0)
	for n > 0 {
		n >>= 1
		s++
	}
	return s
}
