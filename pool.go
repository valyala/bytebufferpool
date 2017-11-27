package bytebufferpool

import (
	"sort"
	"sync/atomic"
)

const (
	defaultMinBitSize = 6 // 2**6=64 is a CPU cache line size
	steps             = 20

	defaultMinSize = 1 << defaultMinBitSize
	defaultMaxSize = 1 << (defaultMinBitSize + steps - 1)

	calibrateCallsThreshold = 42000
	maxPercentile           = 0.95
)

// Pool represents byte buffer pool.
//
// Distinct pools may be used for distinct types of byte buffers.
// Properly determined byte buffer types with their own pools may help reducing
// memory waste.
type Pool struct {
	calls       [steps]uint64
	calibrating uint64

	defaultSize uint64
	maxSize     uint64

	minBitSize uint64
	minSize    uint64

	actualPool // Conditional compilation on flag
}

var defaultPool Pool

// Get returns an empty byte buffer from the pool.
//
// Got byte buffer may be returned to the pool via Put call.
// This reduces the number of memory allocations required for byte buffer
// management.
func Get() *ByteBuffer { return defaultPool.Get() }

// Get returns new byte buffer with zero length.
//
// The byte buffer may be returned to the pool via Put after the use
// in order to minimize GC overhead.
func (p *Pool) Get() *ByteBuffer {
	v := p.pool.Get()
	if v != nil {
		b := v.(*ByteBuffer)
		b.Reset()
		return b
	}
	return &ByteBuffer{
		B: make([]byte, 0, atomic.LoadUint64(&p.defaultSize)),
	}
}

// GetLen returns a buufer with its
// []byte slice of the exact len as specified
//
// The byte buffer may be returned to the pool via Put after the use
// in order to minimize GC overhead.
func GetLen(s int) *ByteBuffer { return defaultPool.GetLen(s) }

// GetLen return a buufer with its
// []byte slice of the exact len as specified
//
// The byte buffer may be returned to the pool via Put after the use
// in order to minimize GC overhead.
func (p *Pool) GetLen(s int) *ByteBuffer {
	v := p.pool.Get()
	if v == nil {
		return &ByteBuffer{
			B: make([]byte, s),
		}
	}

	b := v.(*ByteBuffer)
	if cap(b.B) < s {
		// Create a new []byte slice
		b.B = make([]byte, s)
	} else {
		b.B = b.B[:s]
	}
	return b
}

// Put returns byte buffer to the pool.
//
// ByteBuffer.B mustn't be touched after returning it to the pool.
// Otherwise data races will occur.
func Put(b *ByteBuffer) { defaultPool.Put(b) }

// Put releases byte buffer obtained via Get to the pool.
//
// The buffer mustn't be accessed after returning to the pool.
func (p *Pool) Put(b *ByteBuffer) {
	if p.minBitSize == 0 {
		p.initBins()
	}

	idx := index(p.minBitSize, len(b.B))

	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	maxSize := int(atomic.LoadUint64(&p.maxSize))
	if maxSize == 0 || cap(b.B) <= maxSize {
		p.pool.Put(b)
	}
}

func (p *Pool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	if p.minBitSize == 0 {
		p.initBins()
	}

	a := make(callSizes, 0, steps)
	var callsSum uint64
	for i := uint64(0); i < steps; i++ {
		calls := atomic.SwapUint64(&p.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,
			size:  p.minSize << i,
		})
	}
	if p.minBitSize+steps < 32 && a[steps-1].calls > a[0].calls {
		// Increase the first bin's size
		p.resizeBins(p.minBitSize + 1)
	} else if p.minBitSize > defaultMinBitSize &&
		a[0].calls > 0 &&
		a[steps-2].calls == 0 &&
		a[steps-1].calls == 0 {
		// Decrease the size of first bin's size
		p.resizeBins(p.minBitSize - 1)
	}
	sort.Sort(a)

	defaultSize := a[0].size
	maxSize := defaultSize

	maxSum := uint64(float64(callsSum) * maxPercentile)
	callsSum = 0
	for i := 0; i < steps; i++ {
		if callsSum > maxSum {
			break
		}
		callsSum += a[i].calls
		size := a[i].size
		if size > maxSize {
			maxSize = size
		}
	}

	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)

	atomic.StoreUint64(&p.calibrating, 0)
}

func (p *Pool) resizeBins(minBitSize uint64) {
	atomic.StoreUint64(&p.minBitSize, minBitSize)
	atomic.StoreUint64(&p.minSize, 1<<minBitSize)
}

func (p *Pool) initBins() {
	atomic.StoreUint64(&p.minBitSize, defaultMinBitSize)
	atomic.StoreUint64(&p.minSize, 1<<defaultMinBitSize)
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

func index(minBitSize uint64, n int) int {
	n--
	n >>= minBitSize
	idx := 0
	for n > 0 {
		n >>= 1
		idx++
	}
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}
