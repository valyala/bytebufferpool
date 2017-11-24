// +build !shortlivedpool

package bytebufferpool

import (
	"log"
	"sync"
)

type actualPool struct {
	pool sync.Pool
}

func init() {
	log.Println("Using standard sync.pool")
}
