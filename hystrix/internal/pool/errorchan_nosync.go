//go:build !synctest

package pool

import (
	"sync"
)

var singleErrorChanPool = sync.Pool{
	New: func() any {
		return make(chan error, 1)
	},
}

func AcquireSingleErrorChan() chan error {
	c := singleErrorChanPool.Get()
	if errchan, ok := c.(chan error); ok {
		return errchan
	}

	return make(chan error, 1)
}

func ReleaseSingleErrorChan(errchan chan error) {
	singleErrorChanPool.Put(errchan)
}
