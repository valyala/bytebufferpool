package bytebufferpool

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"
)

func TestByteBufferReadFrom(t *testing.T) {
	prefix := "foobar"
	expectedS := "asadfsdafsadfasdfisdsdfa"
	prefixLen := int64(len(prefix))
	expectedN := int64(len(expectedS))

	var bb ByteBuffer
	bb.WriteString(prefix)

	rf := (io.ReaderFrom)(&bb)
	for i := 0; i < 20; i++ {
		r := bytes.NewBufferString(expectedS)
		n, err := rf.ReadFrom(r)
		if n != expectedN {
			t.Fatalf("unexpected n=%d. Expecting %d. iteration %d", n, expectedN, i)
		}
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		bbLen := int64(bb.Len())
		expectedLen := prefixLen + int64(i+1)*expectedN
		if bbLen != expectedLen {
			t.Fatalf("unexpected byteBuffer length: %d. Expecting %d", bbLen, expectedLen)
		}
		for j := 0; j < i; j++ {
			start := prefixLen + int64(j)*expectedN
			b := bb.B[start : start+expectedN]
			if string(b) != expectedS {
				t.Fatalf("unexpected byteBuffer contents: %q. Expecting %q", b, expectedS)
			}
		}
	}
}

func TestByteBufferWriteTo(t *testing.T) {
	expectedS := "foobarbaz"
	var bb ByteBuffer
	bb.WriteString(expectedS[:3])
	bb.WriteString(expectedS[3:])

	wt := (io.WriterTo)(&bb)
	var w bytes.Buffer
	for i := 0; i < 10; i++ {
		n, err := wt.WriteTo(&w)
		if n != int64(len(expectedS)) {
			t.Fatalf("unexpected n returned from WriteTo: %d. Expecting %d", n, len(expectedS))
		}
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		s := string(w.Bytes())
		if s != expectedS {
			t.Fatalf("unexpected string written %q. Expecting %q", s, expectedS)
		}
		w.Reset()
	}
}

func TestByteBufferGetPutSerial(t *testing.T) {
	testByteBufferGetPut(t)
}

func TestByteBufferGetPutConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testByteBufferGetPut(t)
			ch <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout!")
		}
	}
}

func testByteBufferGetPut(t *testing.T) {
	for i := 0; i < 10; i++ {
		expectedS := fmt.Sprintf("num %d", i)
		b := Get()
		b.B = append(b.B, "num "...)
		b.B = append(b.B, fmt.Sprintf("%d", i)...)
		if string(b.B) != expectedS {
			t.Fatalf("unexpected result: %q. Expecting %q", b.B, expectedS)
		}
		Put(b)
	}
}

func testByteBufferGetString(t *testing.T) {
	for i := 0; i < 10; i++ {
		expectedS := fmt.Sprintf("num %d", i)
		b := Get()
		b.SetString(expectedS)
		if b.String() != expectedS {
			t.Fatalf("unexpected result: %q. Expecting %q", b.B, expectedS)
		}
		Put(b)
	}
}

func TestByteBufferGetStringSerial(t *testing.T) {
	testByteBufferGetString(t)
}

func TestByteBufferGetStringConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testByteBufferGetString(t)
			ch <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout!")
		}
	}
}

func TestByteBufferReadFromEmpty(t *testing.T) {
	var bb ByteBuffer
	r := bytes.NewBuffer(nil)
	n, err := bb.ReadFrom(r)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if n != 0 {
		t.Fatalf("unexpected n=%d. Expecting 0", n)
	}
	if cap(bb.B) < 64 {
		t.Fatalf("unexpected capacity %d; expected at least 64", cap(bb.B))
	}
	if bb.Len() != 0 {
		t.Fatalf("unexpected len %d; expected 0", bb.Len())
	}
}

func TestByteBufferReadFromSmallInitialCap(t *testing.T) {
	for initCap := 1; initCap < 64; initCap++ {
		var bb ByteBuffer
		bb.B = make([]byte, initCap)
		for j := 0; j < initCap; j++ {
			bb.B[j] = 'x'
		}
		expectedS := "hello world from reader"
		r := bytes.NewBufferString(expectedS)
		n, err := bb.ReadFrom(r)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if n != int64(len(expectedS)) {
			t.Fatalf("unexpected n=%d. Expecting %d", n, len(expectedS))
		}
		if cap(bb.B) < 64 {
			t.Fatalf("initCap=%d: unexpected cap=%d. Expecting at least 64", initCap, cap(bb.B))
		}
		expectedTotal := string(bytes.Repeat([]byte{'x'}, initCap)) + expectedS
		if string(bb.B) != expectedTotal {
			t.Fatalf("unexpected result %q; expecting %q", string(bb.B), expectedTotal)
		}
	}
}

func TestByteBufferReadFromReallocSliceUpdate(t *testing.T) {
	var bb ByteBuffer
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	r := bytes.NewReader(data)
	n, err := bb.ReadFrom(r)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("unexpected n=%d. Expecting %d", n, len(data))
	}
	if !bytes.Equal(bb.B, data) {
		t.Fatalf("read data does not match original")
	}
	if bb.Len() != len(data) {
		t.Fatalf("bb.Len()=%d; expecting %d", bb.Len(), len(data))
	}
}

type errReader struct {
	data []byte
	err  error
}

func (er *errReader) Read(p []byte) (int, error) {
	if len(er.data) == 0 {
		return 0, er.err
	}
	n := copy(p, er.data)
	er.data = er.data[n:]
	return n, er.err
}

func TestByteBufferReadFromErrorPreservesRead(t *testing.T) {
	customErr := fmt.Errorf("read failure")
	r := &errReader{
		data: []byte("partial content"),
		err:  customErr,
	}

	var bb ByteBuffer
	n, err := bb.ReadFrom(r)
	if err != customErr {
		t.Fatalf("expected %v, got %v", customErr, err)
	}
	if n != int64(len("partial content")) {
		t.Fatalf("expected n=%d, got %d", len("partial content"), n)
	}
	if bb.String() != "partial content" {
		t.Fatalf("expected string %q, got %q", "partial content", bb.String())
	}
}

