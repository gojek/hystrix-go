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
		for _, x := range []float64{10, 11, 9} {
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
		for _, x := range []float64{0.5, 1.5, 2.5, 3.5, 4.5} {
			n.Increment(x)
			time.Sleep(1 * time.Second)
		}

		if val := n.Avg(time.Now()); val != 1.25 {
			t.Errorf("Avg returned %v instead of 1.25", val)
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
		n.UpdateMax(float64(i))
	}
}
