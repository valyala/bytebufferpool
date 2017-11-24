// +build !shortlivedpool

package bytebufferpool

import (
	"log"
	"sync"
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

	pool sync.Pool
}

func init() {
	log.Println("Using standard sync.pool")
}
