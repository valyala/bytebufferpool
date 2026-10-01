package bytebufferpool

import (
	"bytes"
	"testing"
)

func BenchmarkByteBufferWrite(b *testing.B) {
	s := []byte("foobarbaz")
	b.RunParallel(func(pb *testing.PB) {
		var buf ByteBuffer
		for pb.Next() {
			for i := 0; i < 100; i++ {
				buf.Write(s)
			}
			buf.Reset()
		}
	})
}

func BenchmarkBytesBufferWrite(b *testing.B) {
	s := []byte("foobarbaz")
	b.RunParallel(func(pb *testing.PB) {
		var buf bytes.Buffer
		for pb.Next() {
			for i := 0; i < 100; i++ {
				buf.Write(s)
			}
			buf.Reset()
		}
	})
}

func BenchmarkPoolGetPut(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bb := Get()
			bb.B = append(bb.B, "foobarbaz"...)
			Put(bb)
		}
	})
}

func BenchmarkPoolGetBytesPutBytes(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := GetBytes()
			buf = append(buf, "foobarbaz"...)
			PutBytes(buf)
		}
	})
}
