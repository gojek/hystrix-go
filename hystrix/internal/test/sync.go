//go:build synctest

package test

import (
	"testing"
	"testing/synctest"
)

func SyncTest(t *testing.T, f func(t *testing.T)) {
	synctest.Test(t, func(t *testing.T) {
		f(t)
		SyncTestWait()
	})
}

func SyncTestWait() {
	synctest.Wait()
}

func SkipSyncTest(t *testing.T, _ func(t *testing.T)) {
	t.Skip(t.Name(), ": skipping sync test")
}
