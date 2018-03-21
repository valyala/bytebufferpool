// +build !shortlivedpool

package bytebufferpool

import (
	"sync"
)

type actualPool struct {
	pool sync.Pool
}
