package bytebufferpool

import (
	"math/bits"
	"math/rand"
	"testing"
	"time"
)

func TestIndex(t *testing.T) {
	testIndex(t, 0, 0)
	testIndex(t, 1, 0)

	testIndex(t, minSize-1, 0)
	testIndex(t, minSize, 0)
	testIndex(t, minSize+1, 1)

	testIndex(t, 2*minSize-1, 1)
	testIndex(t, 2*minSize, 1)
	testIndex(t, 2*minSize+1, 2)

	testIndex(t, maxSize-1, steps-1)
	testIndex(t, maxSize, steps-1)
	testIndex(t, maxSize+1, steps-1)
}

func testIndex(t *testing.T, n, expectedIdx int) {
	idx := index(n)
	if idx != expectedIdx {
		t.Fatalf("unexpected idx for n=%d: %d. Expecting %d", n, idx, expectedIdx)
	}
}

func TestPoolCalibrate(t *testing.T) {
	for i := 0; i < steps*calibrateCallsThreshold; i++ {
		n := 1004
		if i%15 == 0 {
			n = rand.Intn(15234)
		}
		testGetPut(t, n)
	}
}

func TestPoolCalibrateWithAdjustment(t *testing.T) {

	var p Pool

	const n = 510

	adjN := n << 2

	// smaller buffer
	allocNBytesMtimes(&p, n, calibrateCallsThreshold-10)

	// t.Log(p.calls)

	// never trigger calibrate, never used as adjustment for defaultSize
	for i, s := 0, adjN<<4; i < calibrateCallsThreshold>>1; i++ {
		v := s + rand.Intn(maxSize)
		allocNBytesInP(&p, v)
	}

	// larger buffer
	allocNBytesMtimes(&p, adjN, calibrateCallsThreshold-10)

	// t.Log(p.calls)

	// now throw away existing larger buf from pool
	_ = p.Get()

	// ... and now finish with new smaller buf (emulate a long process that uses it)
	allocNBytesMtimes(&p, n, 11)

	// t.Logf("%#v", p)

	if v := powOfTwo64(uint64(adjN)); v != p.defaultSize {
		t.Fatalf("wrong pool final defaultSize: want %d, got %d", v, p.defaultSize)
	}
}

func TestPoolVariousSizesSerial(t *testing.T) {
	testPoolVariousSizes(t)
}

func TestPoolVariousSizesConcurrent(t *testing.T) {
	concurrency := 5
	ch := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		go func() {
			testPoolVariousSizes(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout")
		}
	}
}

//go:noinline
func TestIntArithmetic(t *testing.T) {

	if float64(maxPercentileNumer) != (float64(maxPercentile) * float64(maxPercentileDenom)) {
		t.Fatalf("wrong maxPercentile interpolation: want %f, got %f", maxPercentile, float64(maxPercentileNumer)/float64(maxPercentileDenom))
	}

	if float64(calibrateDefaultSizeAdjustmentsFactorNumer) != (float64(calibrateDefaultSizeAdjustmentsFactor) * float64(calibrateDefaultSizeAdjustmentsFactorDenom)) {
		t.Fatalf("wrong maxPercentile interpolation: want %f, got %f", calibrateDefaultSizeAdjustmentsFactor, float64(calibrateDefaultSizeAdjustmentsFactorNumer)/float64(calibrateDefaultSizeAdjustmentsFactorDenom))
	}
}

func testPoolVariousSizes(t *testing.T) {
	for i := 0; i < steps+1; i++ {
		n := (1 << uint32(i))

		testGetPut(t, n)
		testGetPut(t, n+1)
		testGetPut(t, n-1)

		for j := 0; j < 10; j++ {
			testGetPut(t, j+n)
		}
	}
}

func testGetPut(t *testing.T, n int) {
	bb := Get()
	if len(bb.B) > 0 {
		t.Fatalf("non-empty byte buffer returned from acquire")
	}
	bb.B = allocNBytes(bb.B, n)
	Put(bb)
}

func allocNBytes(dst []byte, n int) []byte {
	diff := n - cap(dst)
	if diff <= 0 {
		return dst[:n]
	}
	// must return buffer with len == requested size n, not `n - cap(dst)`
	return append(dst[:cap(dst)], make([]byte, diff)...)
}

func allocNBytesInP(p *Pool, n int) {
	b := p.Get()
	b.B = allocNBytes(b.B, n)
	p.Put(b)
}

func allocNBytesMtimes(p *Pool, n, limit int) {
	for i := 0; i < limit; i++ {
		allocNBytesInP(p, n)
	}
}

// 2^z >= n with min(z)
func powOfTwo64(n uint64) uint64 {
	// ((n - 1) & n) - remove the leftmost one bit, 2^k ==> 0, 0 ==> 0, others > 0
	// ((n - 1) & n) >> 1 - place for sign to avoid overflow, 2^k ==> 0, 0 ==> 0, others > 0
	// ^(((n - 1) & n) >> 1) - invert result, 2^k ==> uint64(-1), 0 ==> uint64(-1), others < -1
	// (^(((n - 1) & n) >> 1) + 1) - for 2^k ==> 0, 0 ==> 0, others < 0
	// uint(^(((n - 1) & n) >> 1) + 1 - z) >> 63 - got sign of result as leftmost bit, 2^k -> 0, 0 -> 0, others -> 1
	a := int(uint64(^(((n-1)&n)>>1)+1) >> 63)
	z := int(((n - 1) &^ n) >> 63) // 0 -> 1, others -> 0
	return 1 << uint(bits.Len64(n)-1+z+a)
}

func allocNMBytesInP(p *Pool, n, m int) {
	// ATN! preserve order, its important
	bn := p.Get()
	bm := p.Get()
	bn.B = allocNBytes(bn.B, n)
	bm.B = allocNBytes(bm.B, m)
	p.Put(bn)
	p.Put(bm)
}

func allocNMBytesXtimes(p *Pool, n, m int, limit int) {
	for i := 0; i < limit; i++ {
		allocNMBytesInP(p, n, m)
	}
}
