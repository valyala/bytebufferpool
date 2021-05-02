package bytebufferpool

import (
	"sort"
	"sync"
	"sync/atomic"
)

const (
	minBitSize = 6 // 2**6=64 is a CPU cache line size
	steps      = 20

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)

	calibrateCallsThreshold = 42000
	maxPercentile           = 0.95

	callsSumMaxValue = steps * calibrateCallsThreshold

	fractionDenominator = uint64(100) // denominator of regular fractions

	// regular fraction of maxPercentile
	maxPercentileRNumer = uint64(maxPercentile * float64(fractionDenominator)) // numerator of maxPercentile
	maxPercentileGcd    = uint64(5)                                            // gcd(maxPercentileRNumer, fractionDenominator) = gcd(int(maxPercentile * 100), 100)
	maxPercentileNumer  = maxPercentileRNumer / maxPercentileGcd               // simplified numerator of maxPercentile
	maxPercentileDenom  = fractionDenominator / maxPercentileGcd               // simplified denominator of maxPercentile

	// allowable size spread for DefaultSize additional adjustment
	calibrateDefaultSizeAdjustmentsSpread = 0.05                                      // down to 5% of initial DefaultSize` calls count
	calibrateDefaultSizeAdjustmentsFactor = 1 - calibrateDefaultSizeAdjustmentsSpread // see calibrate() below

	// regular fraction of calibrateDefaultSizeAdjustmentsFactor
	calibrateDefaultSizeAdjustmentsFactorRNumer = uint64(calibrateDefaultSizeAdjustmentsFactor * float64(fractionDenominator)) // numerator of calibrateDefaultSizeAdjustmentsFactor
	calibrateDSASGcd                            = uint64(5)                                                                    // gcd(calibrateDefaultSizeAdjustmentsFactorRNumer, fractionDenominator)
	calibrateDefaultSizeAdjustmentsFactorNumer  = calibrateDefaultSizeAdjustmentsFactorRNumer / calibrateDSASGcd               // simplified numerator of calibrateDefaultSizeAdjustmentsFactor
	calibrateDefaultSizeAdjustmentsFactorDenom  = fractionDenominator / calibrateDSASGcd                                       // simplified denominator of calibrateDefaultSizeAdjustmentsFactor
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

	pool sync.Pool
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
		return v.(*ByteBuffer)
	}
	return &ByteBuffer{
		B: make([]byte, 0, atomic.LoadUint64(&p.defaultSize)),
	}
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
	idx := index(len(b.B))

	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	maxSize := int(atomic.LoadUint64(&p.maxSize))
	if maxSize == 0 || cap(b.B) <= maxSize {
		b.Reset()
		p.pool.Put(b)
	}
}

func (p *Pool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	a := make(callSizes, 0, steps)

	callsSum := uint64(0)

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

	// callsSum <= steps * calibrateCallsThreshold + maybe small R = callsSumMaxValue + R <<<< (MaxUint64 / fractionDenominator),
	// maxPercentileNumer < fractionDenominator, therefore, integer multiplication by a fraction can be used without overflow
	maxSum := (callsSum * maxPercentileNumer) / maxPercentileDenom // == uint64(callsSum * maxPercentile)

	// avoid visiting a[0] one more times in `for` loop below
	callsSum = a[0].calls

	// defaultSize adjust cond:
	//     ( abs(a[0].calls - a[i].calls) < a[0].calls * calibrateDefaultSizeAdjustmentsSpread ) && ( defaultSize < a[i].size )
	// due to fact that a is sorted by calls desc,
	// abs(a[0].calls - a[i].calls) === a[0].calls - a[i].calls ==>
	// a[0].calls - a[i].calls < a[0].calls * calibrateDefaultSizeAdjustmentsSpread ==>
	// a[0].calls - a[0].calls * calibrateDefaultSizeAdjustmentsSpread < a[i].calls ==>
	// a[i].calls > a[0].calls * (1 - calibrateDefaultSizeAdjustmentsSpread) ==>
	// a[i].calls > a[0].calls * calibrateDefaultSizeAdjustmentsFactor
	// and we can pre-calculate a[0].calls * calibrateDefaultSizeAdjustmentsFactor

	// a[0].calls ~= calibrateCallsThreshold + maybe small R <<<< (MaxUint64 / fractionDenominator)
	defSizeAdjustCallsThreshold := (a[0].calls * calibrateDefaultSizeAdjustmentsFactorNumer) / calibrateDefaultSizeAdjustmentsFactorDenom // == uint64(a[0].calls * calibrateDefaultSizeAdjustmentsFactor)

	for i := 1; i < steps; i++ {

		if callsSum > maxSum {
			break
		}

		size := a[i].size

		if (a[i].calls > defSizeAdjustCallsThreshold) && (size > defaultSize) {
			defaultSize = size
		}

		if size > maxSize {
			maxSize = size
		}

		callsSum += a[i].calls
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

func index(n int) int {
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
