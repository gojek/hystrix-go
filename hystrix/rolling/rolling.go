package rolling

import (
	"sync/atomic"
	"time"
)

const numberBucketSize = 10 + 1

// Number tracks a numberBucket over a bounded number of
// time buckets. Currently the buckets are one second long and only the last 10 seconds are kept.
type Number struct {
	buckets []atomic.Pointer[numberBucket]
}

type numberBucket struct {
	key   int64
	value atomic.Int64
}

// NewNumber initializes a RollingNumber struct.
func NewNumber() *Number {
	buckets := make([]atomic.Pointer[numberBucket], numberBucketSize)
	for i := 0; i < numberBucketSize; i++ {
		buckets[i].Store(&numberBucket{})
	}
	return &Number{
		buckets: buckets,
	}
}

func (r *Number) getBucketAt(t time.Time) *numberBucket {
	epoch := t.Unix()
	for {
		bucket := r.buckets[epoch%numberBucketSize].Load()
		if bucket.key == epoch {
			return bucket
		}

		newBucket := &numberBucket{key: epoch}
		if r.buckets[epoch%numberBucketSize].CompareAndSwap(bucket, newBucket) { // Swap successful
			return newBucket
		}
	}
}

// Increment increments the number in current timeBucket.
func (r *Number) Increment(i int64) {
	r.IncrementAt(time.Now(), i)
}

// IncrementAt increments the number in current timeBucket.
// Note: Caller is responsible to pass the current time
func (r *Number) IncrementAt(t time.Time, i int64) {
	if i == 0 {
		return
	}

	r.getBucketAt(t).value.Add(i)
}

// UpdateMax updates the maximum value in the current bucket.
func (r *Number) UpdateMax(n int64) {
	r.UpdateMaxAt(time.Now(), n)
}

// UpdateMaxAt updates the maximum value in the current bucket.
// Note: Caller is responsible to pass the current time
func (r *Number) UpdateMaxAt(t time.Time, n int64) {
	r.getBucketAt(t).updateMax(n)
}

// Sum sums the values over the buckets in the last 10 seconds.
func (r *Number) Sum(now time.Time) int64 {
	minKey := now.Unix() - 10
	sum := int64(0)

	for i := range r.buckets {
		if bucket := r.buckets[i].Load(); bucket.key >= minKey {
			sum += bucket.value.Load()
		}
	}

	return sum
}

// Max returns the maximum value seen in the last 10 seconds.
func (r *Number) Max(now time.Time) int64 {
	minKey := now.Unix() - 10
	var maxVal int64

	for i := range r.buckets {
		if bucket := r.buckets[i].Load(); bucket.key >= minKey {
			if val := bucket.value.Load(); val > maxVal {
				maxVal = val
			}
		}
	}

	return maxVal
}

func (r *Number) Avg(now time.Time) float64 {
	return float64(r.Sum(now)) / (numberBucketSize - 1)
}

func (r *Number) Reset() {
	for i := range r.buckets {
		r.buckets[i].Store(&numberBucket{})
	}
}

func (b *numberBucket) updateMax(n int64) {
	for {
		v := b.value.Load()
		if n <= v {
			return
		}
		if b.value.CompareAndSwap(v, n) {
			return
		}
	}
}
