package rolling

import (
	"testing"
	"time"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func TestMax(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		n := NewNumber()
		for _, x := range []int64{10, 11, 9} {
			n.UpdateMax(x)
			time.Sleep(1 * time.Second)
		}

		if val := n.Max(time.Now()); val != 11 {
			t.Errorf("Max returned %v instead of 11", val)
		}
	})
}

func TestAvg(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		n := NewNumber()
		for _, x := range []int64{5, 15, 25, 35, 45} {
			n.Increment(x)
			time.Sleep(1 * time.Second)
		}

		if val := n.Avg(time.Now()); val != 12.5 {
			t.Errorf("Avg returned %v instead of 12.5", val)
		}
	})
}

func BenchmarkRollingNumberIncrement(b *testing.B) {
	n := NewNumber()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.Increment(1)
	}
}

func BenchmarkRollingNumberUpdateMax(b *testing.B) {
	n := NewNumber()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n.UpdateMax(int64(i))
	}
}
