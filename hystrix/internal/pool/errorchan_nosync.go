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
	return singleErrorChanPool.Get().(chan error)
}

func ReleaseSingleErrorChan(errchan chan error) {
	if errchan == nil {
		return
	}
	singleErrorChanPool.Put(errchan)
}
