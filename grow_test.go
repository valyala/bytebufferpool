package bytebufferpool

import "testing"

func TestGrow(t *testing.T) {
	var b ByteBuffer
	b.Grow(64)
	if cap(b.B) < 64 {
		t.Fatalf("cap=%d", cap(b.B))
	}
	b.WriteString("hello")
	if b.String() != "hello" {
		t.Fatal(b.String())
	}
	// Growing again should not drop content
	b.Grow(100)
	if b.String() != "hello" {
		t.Fatal(b.String())
	}
	if cap(b.B)-len(b.B) < 100 {
		t.Fatalf("remaining cap=%d", cap(b.B)-len(b.B))
	}
}
