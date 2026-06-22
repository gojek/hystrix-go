package rolling

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestOrdinal(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testOrdinal(t)
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testOrdinal(t)
			synctest.Wait()
		})
	})
}

func testOrdinal(t *testing.T) {
	r := NewTiming()
	if r.Mean() != 0 {
		t.Fatalf("Mean should be 0, but got %v", r.Mean())
	}

	// and given a set of lengths and percentiles"
	var ordinalTests = []struct {
		length   int
		perc     float64
		expected int64
	}{
		{1, 0, 1},
		{2, 0, 1},
		{2, 50, 1},
		{2, 51, 2},
		{5, 30, 2},
		{5, 40, 2},
		{5, 50, 3},
		{11, 25, 3},
		{11, 50, 6},
		{11, 75, 9},
		{11, 100, 11},
	}

	for _, s := range ordinalTests {
		if val := r.ordinal(s.length, s.perc); val != s.expected {
			t.Errorf("ordinal(%v, %v) returned %v instead of %v", s.length, s.perc, val, s.expected)
		}
	}

	// after adding 2 timings
	r.Add(100 * time.Millisecond)
	time.Sleep(2 * time.Second)
	r.Add(200 * time.Millisecond)

	if val := r.Mean(); val != 150 {
		t.Fatalf("Mean returned %v instead of %v", val, 150)
	}

	// after adding many timings
	durations := []int{1, 1004, 1004, 1004, 1004, 1004, 1004, 1004, 1004, 1004, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1005, 1006, 1006, 1006, 1006, 1007, 1007, 1007, 1008, 1015}
	for _, d := range durations {
		r.Add(time.Duration(d) * time.Millisecond)
	}
	time.Sleep(time.Second)

	if val := r.Percentile(0); val != 1 {
		t.Errorf("Percentile returned %v instead of 1", val)
	}
	if val := r.Percentile(75); val != 1006 {
		t.Errorf("Percentile returned %v instead of 1006", val)
	}
	if val := r.Percentile(99); val != 1015 {
		t.Errorf("Percentile returned %v instead of 1015", val)
	}
	if val := r.Percentile(100); val != 1015 {
		t.Errorf("Percentile returned %v instead of 1015", val)
	}
}
