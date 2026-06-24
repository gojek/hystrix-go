//go:build !synctest

package test

import (
	"testing"
)

func SyncTest(t *testing.T, f func(t *testing.T)) {
	f(t)
}

func SyncTestWait() {}

func SkipSyncTest(t *testing.T, f func(t *testing.T)) {
	f(t)
}
