//go:build synctest

package pool

// avoid reusing channel across synctest bubble
func AcquireSingleErrorChan() chan error {
	return make(chan error, 1)
}

func ReleaseSingleErrorChan(_ chan error) {}
