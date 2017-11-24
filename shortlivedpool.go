// +build shortlivedpool

package bytebufferpool

import (
	"log"

	"github.com/gallir/shortlivedpool"
)

type actualPool struct {
	pool shortlivedpool.Pool
}

func init() {
	log.Println("Using github.com/gallir/shortlivedpool instead of sync.pool")
}
