package bytebufferpool

import (
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
	return append(dst, make([]byte, diff)...)
}

func TestPoolGetBytesSerial(t *testing.T) {
	for i := 0; i < 100; i++ {
		b := GetBytes()
		if len(b) > 0 {
			t.Fatalf("unexpected non-empty slice: len=%d", len(b))
		}
		expected := "hello world"
		b = append(b, expected...)
		if string(b) != expected {
			t.Fatalf("unexpected content: %q, expecting %q", string(b), expected)
		}
		PutBytes(b)
	}
}

func TestPoolGetBytesConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func(worker int) {
			for j := 0; j < 50; j++ {
				b := GetBytes()
				if len(b) > 0 {
					t.Errorf("worker %d: non-empty slice returned: len=%d", worker, len(b))
					ch <- struct{}{}
					return
				}
				b = append(b, "test-data"...)
				PutBytes(b)
			}
			ch <- struct{}{}
		}(i)
	}
	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout")
		}
	}
}

func TestPoolGetSliceSerial(t *testing.T) {
	for i := 0; i < 100; i++ {
		b := GetSlice()
		if len(b) > 0 {
			t.Fatalf("unexpected non-empty slice: len=%d", len(b))
		}
		expected := "slice data"
		b = append(b, expected...)
		if string(b) != expected {
			t.Fatalf("unexpected content: %q, expecting %q", string(b), expected)
		}
		PutSlice(b)
	}
}

func TestPoolGetSliceConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				b := GetSlice()
				if len(b) > 0 {
					t.Errorf("non-empty slice returned: len=%d", len(b))
					ch <- struct{}{}
					return
				}
				b = append(b, "slice-data"...)
				PutSlice(b)
			}
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

func TestCustomPoolBytes(t *testing.T) {
	var p Pool
	b := p.GetBytes()
	if len(b) > 0 {
		t.Fatalf("expected empty slice, got len=%d", len(b))
	}
	b = append(b, "custom-pool"...)
	p.PutBytes(b)

	s := p.GetSlice()
	if len(s) > 0 {
		t.Fatalf("expected empty slice, got len=%d", len(s))
	}
	s = append(s, "custom-slice"...)
	p.PutSlice(s)

	// Verify PutBytes/PutSlice with nil does not panic
	p.PutBytes(nil)
	p.PutSlice(nil)
	PutBytes(nil)
	PutSlice(nil)
}
