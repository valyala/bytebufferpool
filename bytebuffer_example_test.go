package bytebufferpool_test

import (
	"fmt"

	"github.com/valyala/bytebufferpool"
)

func ExampleByteBuffer() {
	bb := bytebufferpool.Get()

	bb.WriteString("first line\n")
	bb.Write([]byte("second line\n"))

	fmt.Printf("bytebuffer contents=%q", bb.Bytes())

	// It is safe to release byte buffer now, since it is
	// no longer used.
	bytebufferpool.Put(bb)
}
